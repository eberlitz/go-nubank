package client

const accountFeedQuery = `{
	viewer {
		savingsAccount {
			id
			feed {
				id
				__typename
				title
				detail
				postDate
				... on TransferInEvent {
					amount
					originAccount {
						name
					}
				}
				... on TransferOutEvent {
					amount
					destinationAccount {
						name
					}
				}
				... on TransferOutReversalEvent {
					amount
				}
				... on BillPaymentEvent {
					amount
				}
				... on DebitPurchaseEvent {
					amount
				}
				... on BarcodePaymentEvent {
					amount
				}
				... on DebitWithdrawalFeeEvent {
					amount
				}
				... on DebitWithdrawalEvent {
					amount
				}    
			}
		}
	}
}`

type AccountFeedResponse struct {
	Data Data `json:"data"`
}

type Data struct {
	Viewer Viewer `json:"viewer"`
}

type Viewer struct {
	SavingsAccount SavingsAccount `json:"savingsAccount"`
}

type SavingsAccount struct {
	ID   string `json:"id"`
	Feed []Feed `json:"feed"`
}

type Feed struct {
	ID                 string    `json:"id"`
	Typename           string    `json:"__typename"`
	Title              string    `json:"title"`
	Detail             string    `json:"detail"`
	PostDate           string    `json:"postDate"`
	Amount             *float64  `json:"amount"`
	DestinationAccount *NAccount `json:"destinationAccount,omitempty"`
	OriginAccount      *NAccount `json:"originAccount"`
}

type NAccount struct {
	Name string `json:"name"`
}
