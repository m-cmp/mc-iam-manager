(REPO LINK IS HERE)[https://github.com/raccoon-mh/mc-iam-manager-tokenValidator-middleware-PoC]

# Echo JWT Authentication Example

This project demonstrates a simple Echo server with JWT authentication using the `iamtokenvalidatorpoc` library.

## Prerequisites

- Go (version 1.15+)
- Echo framework (version 4)
- `iamtokenvalidatorpoc` library

## Installation

1. Clone the repository:
    ```sh
    git clone <repository-url>
    cd <repository-directory>
    ```

2. Install dependencies:
    ```sh
    go get github.com/labstack/echo/v4
    go get github.com/raccoon-mh/iamtokenvalidatorpoc
    ```

## Configuration

Ensure that the MC-IAM-MANAGER's public key endpoint is correctly configured in the `init` function:
```go
func init() {
    err := iamtokenvalidatorpoc.GetPubkeyIamManager("https://example.com:5000/api/auth/certs")
    if err != nil {
        panic(err.Error())
    }
}
```

## Usage

1. Run the server:
    ```sh
    go run main.go
    ```

2. The server will start on `http://localhost:1323`.

### Endpoints

- `GET /`: A public endpoint that returns "Hello, World!".
- `ANY /protected`: A protected endpoint that requires a valid JWT token.

### Middleware

- `isTokenValid`: Validates the JWT token.
- `setUserRole`: Extracts and sets user roles from the token.

### JWT Validation

The JWT token is validated using the `iamtokenvalidatorpoc` library. The token should be included in the `Authorization` header as a Bearer token.

## Example

To access the protected endpoint, include a valid JWT token in the request header:

```sh
curl -H "Authorization: Bearer <your-token>" http://localhost:1323/protected
```
# Echo JWT Authentication Example

This project demonstrates a simple Echo server with JWT authentication using the `iamtokenvalidatorpoc` library.

## Prerequisites

- Go (version 1.15+)
- Echo framework (version 4)
- `iamtokenvalidatorpoc` library

## Installation

1. Clone the repository:
    ```sh
    git clone <repository-url>
    cd <repository-directory>
    ```

2. Install dependencies:
    ```sh
    go get github.com/labstack/echo/v4
    go get github.com/raccoon-mh/iamtokenvalidatorpoc
    ```

## Configuration

Ensure that the MC-IAM-MANAGER's public key endpoint is correctly configured in the `init` function:
```go
func init() {
    err := iamtokenvalidatorpoc.GetPubkeyIamManager("https://example.com:5000/api/auth/certs")
    if err != nil {
        panic(err.Error())
    }
}
```

## Usage

1. Run the server:
    ```sh
    go run main.go
    ```

2. The server will start on `http://localhost:1323`.

### Endpoints

- `GET /`: A public endpoint that returns "Hello, World!".
- `ANY /protected`: A protected endpoint that requires a valid JWT token.

### Middleware

- `isTokenValid`: Validates the JWT token.
- `setUserRole`: Extracts and sets user roles from the token.

### JWT Validation

The JWT token is validated using the `iamtokenvalidatorpoc` library. The token should be included in the `Authorization` header as a Bearer token.

## Example

To access the protected endpoint, include a valid JWT token in the request header:

```sh
curl -H "Authorization: Bearer <your-token>" http://localhost:1323/protected
```