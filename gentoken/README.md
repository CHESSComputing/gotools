# gentoken

`gentoken` is a command-line tool written in Go that generates [JWT (JSON Web
Tokens)](https://jwt.io/) with configurable standard and custom claims. It
supports both **HS256 (HMAC)** and **RS256 (RSA)** signing algorithms.

---

## Features

- Supports **HS256** (with shared secret) and **RS256** (with RSA private key)
- Set standard claims: `iss`, `sub`, `aud`, `exp`, `iat`
- Add complex **custom claims** under the `custom_claims` field using JSON

---

## Installation

1. Clone the repo and build:

```
git clone https://github.com/your-org/gentoken.git
cd gentoken
make
```

---

## Usage

```
./gentoken [flags]
```

### Required Flags (per algorithm)

* For `HS256`:
  `-secret <string>`

* For `RS256`:
  `-privatekey <path-to-private-key.pem>`

### Standard Claim Flags

| Flag   | Description                                  |
| ------ | -------------------------------------------- |
| `-alg` | Signing algorithm (`HS256`, `RS256`)         |
| `-iss` | Issuer (`iss` claim)                         |
| `-sub` | Subject (`sub` claim)                        |
| `-aud` | Audience (`aud` claim), comma-separated list |
| `-exp` | Expiration time in minutes (default: `60`)   |

### Custom Claims

| Flag      | Description                                    |
| --------- | ---------------------------------------------- |
| `-claims` | JSON string inserted under `custom_claims` key |

---

## Examples

### HS256 Example

```
./gentoken \
  -alg=HS256 \
  -secret="mysecret" \
  -iss="MyAuthServer" \
  -sub="user123" \
  -aud="app1,app2" \
  -claims='{"user":"test","scope":"read","roles":["admin"]}'
```

### RS256 Example

```
# generate private key
openssl genrsa -out private.key 2048

# generate public key
openssl rsa -in private.key -pubout > public.key

./gentoken \
  -alg=RS256 \
  -privatekey=private.key \
  -iss="gentoken CLI" \
  -sub="2b4b7431938a482495607486861b2645" \
  -aud="5a8f3bc3fc194f87902d9e6f9b64b7e4" \
  -claims='{
    "user":"test",
    "scope":"read+write",
    "kind":"selftoken",
    "roles":null,
    "application":"gentoken"
  }'
```

---

## Output

The tool prints the signed JWT token to `stdout`:

```
Generated JWT:
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

You can decode the JWT at [jwt.io](https://jwt.io/)

---

## Integration with FOXDEN
The FOXDEN relies on server credentials for token verification. To generate
token with server side client credentials please use the following example:

```
# please replace -secret value with ClientId: part of FOXDEN Authz configuration
./gentoken \
  -alg=HS256 \
  -secret=my-client-id-123 \
  -iss="gentoken CLI" \
  -claims='{
    "user":"test",
    "scope":"read+write",
    "kind":"selftoken",
    "roles":null,
    "application":"gentoken"
  }'
```

Here you need to replace `-secret=...` value with one used by your
`~/.foxden.yaml` configuration file in `Authz->ClientId` section, e.g.
look out the following part of your configuration file:
```
...
Authz:
   ClientId: my-client-id-123
...
```
and use the same clientid in your secret for `gentoken` option above.

---

## Notes

* `iat` and `exp` are set automatically based on current time and `-exp` duration.
* Custom claims are namespaced under `custom_claims` to avoid conflicts with standard claims.
* To pass complex JSON for `-claims`, escape it or use a single-quoted string in the shell.

---

## License

MIT License

