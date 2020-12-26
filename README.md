# go-nubank

Unofficial Nubank Client API for Go

Based on https://github.com/andreroggeri/pynubank

## Remarks

* **Currently only supports authentication with certificates and can only get the list of events from your account**

## Generating certificate for authentication

You can use the this package as a cli to generate the certificate, like:

```sh
go run cmd/nubank-cli/main.go
```

## Usage

```go
import "github.com/eberlitz/go-nubank/pkg/client"

c, err := client.New(client.WithCertificate("./path/to/cert.pem", "./path/to/key.pem"))
if err != nil {
    panic(err)
}

ctx := context.Background()
err = c.Authenticate(ctx, "<CPF>", "<PASSWORD>")
if err != nil {
    panic(err)
}

events, err = c.GetEvents(ctx)
if err != nil {
    panic(err)
}
```


## Development

In order to setup your local development environment you will need to set the env variables as seen in `.envrc.example`;

I recommend using [direnv](https://direnv.net/) to load environment variables from a `.envrc` file when you navigate into the project directory.
There is an `.envrc.example` which can be renamed to `.envrc` to set these values - update accordingly your needs;

There is also a `Makefile` in the project for convenience on several workflows. Run `make` or `make help` for details on how to use it.
