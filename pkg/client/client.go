package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Client interface {
	RequestCode(ctx context.Context, login, password, deviceID string) (*Code, error)
	Authenticate(ctx context.Context, cpf, password string) error
	GetEvents(ctx context.Context) (*EventResponse, error)
	GetAccountFeed(ctx context.Context) ([]Feed, error)
}

type client struct {
	httpClient *http.Client
	appLinks   map[string]string
	auth       authResponse
}

// compile-time check that type implements interface.
var _ Client = (*client)(nil)

type Option func(*client) error

func WithCertificate(certPath, keyPath string) Option {
	return func(cli *client) error {
		config := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		config.Certificates = make([]tls.Certificate, 1)
		var err error
		config.Certificates[0], err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			panic(err)
		}

		cli.httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: config,
			},
		}
		return nil
	}
}

func New(opts ...Option) (Client, error) {
	c := &client{
		httpClient: http.DefaultClient,
	}
	var err error
	for _, opt := range opts {
		err = opt(c)
		if err != nil {
			return c, err
		}
	}
	c.appLinks, err = c.discoverLinks()
	return c, err
}

func (c *client) RequestCode(ctx context.Context, cpf, password, deviceID string) (*Code, error) {
	cert := &certificateRequest{
		key1: generateKey(),
		key2: generateKey(),
	}
	cert.payload = map[string]string{
		"login":             cpf,
		"password":          password,
		"public_key":        getPublicKeyPEMStr(cert.key1),
		"public_key_crypto": getPublicKeyPEMStr(cert.key2),
		"model":             fmt.Sprintf("GoNubank Client (%s)", deviceID),
		"device_id":         deviceID,
	}

	req, err := c.newJSONPostRequest(ctx, c.appLinks["gen_certificate"], cert.payload)
	if err != nil {
		return nil, err
	}
	response, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	authenticate := response.Header.Get("Www-Authenticate")
	if response.StatusCode != http.StatusUnauthorized || authenticate == "" {
		return nil, responseError(response)
	}

	authData := parseAuthenticateHeader(authenticate)
	cert.payload["encrypted-code"] = authData["encrypted-code"]

	return &Code{
		request:     cert,
		codeReqData: authData,
		cli:         c,
	}, err
}

func (c *client) Authenticate(ctx context.Context, cpf, password string) error {
	req, err := c.newJSONPostRequest(ctx, c.appLinks["token"], map[string]string{
		"grant_type":    "password",
		"client_id":     "legacy_client_id",
		"client_secret": "legacy_client_secret",
		"login":         cpf,
		"password":      password,
	})
	if err != nil {
		return err
	}
	response, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: Got %d, Want %d", ErrStatusCodeUnexpected, response.StatusCode, http.StatusOK)
	}

	var responseData authResponse
	err = json.NewDecoder(response.Body).Decode(&responseData)
	if err != nil {
		return err
	}

	for k, l := range responseData.Links {
		c.appLinks[k] = l.Href
	}

	c.auth = responseData
	return nil
}

func (c *client) GetEvents(ctx context.Context) (*EventResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.appLinks["events"], nil)
	if err != nil {
		return nil, err
	}

	response, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: Got %d, Want %d", ErrStatusCodeUnexpected, response.StatusCode, http.StatusOK)
	}

	var responseData *EventResponse
	err = json.NewDecoder(response.Body).Decode(&responseData)
	return responseData, err
}

func (c *client) GetAccountFeed(ctx context.Context) ([]Feed, error) {
	href := c.appLinks["ghostflame"]

	req, err := c.newJSONPostRequest(ctx, href, map[string]interface{}{
		"variables": struct{}{},
		"query":     accountFeedQuery,
	})
	if err != nil {
		return nil, err
	}
	response, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: Got %d, Want %d", ErrStatusCodeUnexpected, response.StatusCode, http.StatusOK)
	}

	var responseData *AccountFeedResponse
	err = json.NewDecoder(response.Body).Decode(&responseData)
	if err != nil {
		return nil, err
	}
	return responseData.Data.Viewer.SavingsAccount.Feed, nil
}

func (c *client) discoverLinks() (map[string]string, error) {
	dict := map[string]string{}
	req, err := http.NewRequest(http.MethodGet, discoveryAppURL, nil)
	if err != nil {
		return dict, err
	}
	response, err := c.doRequest(req)
	if err != nil {
		return dict, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return dict, fmt.Errorf("%w: Got %d, Want %d", ErrStatusCodeUnexpected, response.StatusCode, http.StatusOK)
	}
	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return dict, err
	}
	for k, v := range result {
		if s, ok := v.(string); ok {
			dict[k] = s
		} else if sd, ok := v.(map[string]interface{}); ok {
			// dict[k] = s
			for sk, sv := range sd {
				if s, ok := sv.(string); ok {
					dict[fmt.Sprintf("%s.%s", k, sk)] = s
				}
			}
		}
	}
	return dict, nil
}

func (c *client) newJSONPostRequest(ctx context.Context, url string, payload interface{}) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize JSON payload into request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *client) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Correlation-Id", "WEB-APP.pewW9")
	req.Header.Set("User-Agent", "eberlitz/go-nubank Client - https://github.com/eberlitz/go-nubank")
	return c.httpClient.Do(req)
}

func (c *client) doAuthenticatedRequest(req *http.Request) (*http.Response, error) {
	if c.auth.AccessToken == "" {
		return nil, ErrNotAuthenticated
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.auth.AccessToken))
	return c.httpClient.Do(req)
}

func responseError(response *http.Response) error {
	raw, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: [%d] - %s", ErrStatusCodeUnexpected, response.StatusCode, string(raw))
}

type linkRef struct {
	Href string `json:"href"`
}

type authResponse struct {
	RefreshToken  string             `json:"refresh_token"`
	RefreshBefore *time.Time         `json:"refresh_before"`
	AccessToken   string             `json:"access_token"`
	TokenType     string             `json:"token_type"`
	Links         map[string]linkRef `json:"_links"`
}

type EventResponse struct {
	Events     []EventElement     `json:"events"`
	CustomerID string             `json:"customer_id"`
	AsOf       time.Time          `json:"as_of"`
	Links      EventResponseLinks `json:"_links"`
}

type EventElement struct {
	Description      *string     `json:"description,omitempty"`
	Category         Category    `json:"category"`
	Amount           *int64      `json:"amount,omitempty"`
	Time             time.Time   `json:"time"`
	Source           *Source     `json:"source,omitempty"`
	Title            string      `json:"title"`
	AmountWithoutIof *int64      `json:"amount_without_iof,omitempty"`
	Account          *string     `json:"account,omitempty"`
	Details          *Details    `json:"details,omitempty"`
	ID               string      `json:"id"`
	Links            *EventLinks `json:"_links,omitempty"`
	Tokenized        *bool       `json:"tokenized,omitempty"`
	Href             *string     `json:"href,omitempty"`
}

type Details struct {
	Subcategory         Subcategory `json:"subcategory"`
	Lat                 *float64    `json:"lat,omitempty"`
	Lon                 *float64    `json:"lon,omitempty"`
	Fx                  *Fx         `json:"fx,omitempty"`
	Charges             *Charges    `json:"charges,omitempty"`
	ChargebackRequested *bool       `json:"chargeback_requested,omitempty"`
	Tags                []string    `json:"tags"`
}

type Charges struct {
	Count  int64 `json:"count"`
	Amount int64 `json:"amount"`
}

type Fx struct {
	CurrencyOrigin      string  `json:"currency_origin"`
	AmountOrigin        int64   `json:"amount_origin"`
	AmountUsd           int64   `json:"amount_usd"`
	PreciseAmountOrigin string  `json:"precise_amount_origin"`
	PreciseAmountUsd    string  `json:"precise_amount_usd"`
	ExchangeRate        float64 `json:"exchange_rate"`
}

type EventLinks struct {
	Self Updates `json:"self"`
}

type Updates struct {
	Href string `json:"href"`
}

type EventResponseLinks struct {
	Updates Updates `json:"updates"`
}

type Category string

const (
	AccountLimitSet            Category = "account_limit_set"
	AnticipateEvent            Category = "anticipate_event"
	BillFlowClosed             Category = "bill_flow_closed"
	BillFlowOnOneDayToDueDate  Category = "bill_flow_on_one_day_to_due_date"
	BillFlowPaid               Category = "bill_flow_paid"
	CardActivated              Category = "card_activated"
	CustomerDeviceAuthorized   Category = "customer_device_authorized"
	CustomerInvitationsChanged Category = "customer_invitations_changed"
	CustomerPasswordChanged    Category = "customer_password_changed"
	InitialAccountLimit        Category = "initial_account_limit"
	Payment                    Category = "payment"
	RewardsFee                 Category = "rewards_fee"
	RewardsRedemption          Category = "rewards_redemption"
	RewardsSignup              Category = "rewards_signup"
	Transaction                Category = "transaction"
	TransactionReversed        Category = "transaction_reversed"
	Tutorial                   Category = "tutorial"
	Welcome                    Category = "welcome"
)

type Subcategory string

const (
	CardNotPresent Subcategory = "card_not_present"
	CardPresent    Subcategory = "card_present"
	Unknown        Subcategory = "unknown"
)

type Source string

const (
	InstallmentsMerchant Source = "installments_merchant"
	UpfrontForeign       Source = "upfront_foreign"
	UpfrontNational      Source = "upfront_national"
)
