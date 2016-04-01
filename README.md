# JWT Proxy

The JWT proxy is intended to be used as a complementary service for authenticating, and possibly authorizing requests made between services.
There is a forward proxy component, which can be configured to sign outgoing requests to another service, and a reverse proxy component, which can be used to authenticate incoming requests from another service.

## JWT Forward Proxy

The JWT forward proxy is used to sign outgoing requests with a JWT using a private key.

### Features

- Append a JWT `Authorization` header containing claims about which service the request originated from, and who is the intended recipient
- Autogenerate private signing keys, and publish the public portion to a server following the [key server specification](https://github.com/coreos-inc/jwtproxy/blob/master/jwt/keyserver/keyregistry/README.md)
- Ability to use a preshared private key if we don't want to use autogenerated keys (useful when originating service is HA)
- Ability to sign SSL requests using MITM SSL with configurable certificate authority

### Potential Features

- Ability to append static claims via config
- Ability to read dynamic claims out of a request header and turn them into JWT claims

## JWT Reverse Proxy

The JWT reverse proxy is used to verify incoming requests that were signed by the forward proxy.

### Features

- Ability to decode and verify JWT `Authorization` headers on incoming requests
- Ability to verify the signature based on the specified signing key against a public key fetched from a [key server](https://github.com/coreos-inc/jwtproxy/blob/master/jwt/keyserver/keyregistry/README.md)
- Ability to verify from a single issuer using a pre-shared public key (likely only useful for testing)
- Ability to verify SSL requests by doing SSL termination on behalf of the upstream

### Potential Features

- Ability to verify static claims via config
- Ability to parse and write claims as an unforgeable header sent to the upstream
- Load balancing among multiple upstreams
- Ability to dial a unix socket for communication with upstream server

## Usage

Run with:

```bash
jwtproxy --config config.yaml
```

The configuration yaml file contains a `jwtproxy` top level config flag, which allows a single yaml file to be used to configure multiple services. The presence or absence of a signer config or verifier config block will enable the forward and reverse proxy respectively.

```yaml
jwtproxy:
  <Signer Config>
  <Verifier Config>
```

### Signer Config

Configures and enables the JWT forward signing proxy.

```yaml
jwtproxy:
  signer_proxy:
    enabled: <bool|true>

    # Addr at which to bind proxy server
    listen_addr: <string|:8080>
    shutdown_timeout: <time.Duration|1m>

    # Optional key and CA certificate to forge MITM SSL certificates
    ca_key_file: <path|nil>
    ca_crt_file: <path|nil>

    # Certificates to be trusted by the proxy when its MITM mechanism communicates with remotes.
    # This is optional. Specifying it currently replaces system root certificates entirely.
    trusted_certificates: <[]string|system root certificates>

    signer:
      # Signing service name
      issuer: <string|nil>

      # Validity duration
      expiration_time: <time.Duration|5m>

      # How much time skew we allow between signer and verifier
      max_skew: <time.Duration|1m>

      # Length of random nonce values
      nonce_length: <int|32>

      # Registerable private key source type
      private_key:
        type: <string|nil>
        options: <map[string]interface{}>
```

#### Autogenerated Private Key

Configures a private key source which generates key pairs automatically and publishes them to a key server.

```yaml
private_key:
  type: autogenerated
  options:
    # How often we publish a new key
    rotate_every: <time.Duration|12h>

    # Registerable key server and config at which to publish public keys
    key_server:
      type: <string|nil>
      options: <map[string]interface{}>
```

#### Key Registry Key Server

Configures a key server which talks to a server which implements the key registry protocol.

```yaml
key_server:
  type: keyregistry
  options:
    # Base URL from which to access key registry endpoints.
    registry: <string|nil>
```

#### Preshared Private Key

Configures a private key source which simply uses the key files specified.

```yaml
private_key:
  type: preshared
  options:
    # Unique identifier for the private key
    key_id: <string|nil>

    # Location of PEM encoded private key file
    private_key_path: <path|nil>
```

### Verifier Config

Configures and enables the JWT verifying reverse proxy.

```yaml
jwtproxy:
  verifier_proxy:
    enabled: <bool|true>

    # Addr at which to listen for requests
    listen_addr: <string|:8081>
    shutdown_timeout: <time.Duration|1m>

    # Optional PEM private key and certificate files for SSL termination
    key_file: <path|nil>
    crt_file: <path|nil>

    verifier:
      # Upstream server to which to forward requests
      upstream: <string|nil>

      # Required value for audience claim,
      # Usually our advertised protocol and hostname
      audience: <string|nil>

      # How much time skew we allow between signer and verifier
      max_skew: <time.Duration|1m>

      # Maximum total amount of time for which a JWT can be signed
      max_ttl: <time.Duration|5m>

      # Registerable key server type and options used to fetch
      # public keys for verifying signatures
      key_server:
         type: <string|nil>
         options: <map[string]interface{}>

      # Registerable type and options where we track used nonces
      nonce_storage:
        type: <string|nil>
        options: <map[string]interface{}>
```

#### Key Registry Key Server

Configures a key server which fetches public keys from a server which implements the key registry protocol.

```yaml
key_server:
  type: keyregistry
  options:
    # Base URL from which to access key registry endpoints.
    registry: <string|nil>

    # Optional cache config to alleviate load on the key server.
    cache:
      # How long the keys stay valid in the cache
      duration: <time.Duration|10m>

      # How often expired keys are removed from memory
      purge_interval: <time.Duration|1m>
```

#### Preshared Key Server (Testing Only)

Configures a local preshared mock key server which can return one and only one public key. This should probably be used **only for testing**.

```yaml
key_server:
  type: preshared
  options:
    # Configures the only issuer we will allow
    issuer: <string|nil>

    # Unique ID of the only key from which we validate requests
    key_id: <string|nil>

    # File path to the PEM encoded public key to verify signatures
    public_key_path: <path|nil>
```

#### Local Nonce Storage

Configures nonce storage which stores previously seen nonces in a TTL cache in memory.

```yaml
nonce_storage:
  type: local
  options:
    # How often we run the cache janitor to clean up expired nonces
    purge_interval: <time.Duration|0>
```


### Generate keys

## Generate forward proxy's CA certificate and private key

When it comes to sign HTTPs requests, the forward proxy must *hijack* connections and act as a man-in-the-middle.
Therefore, the proxy requires a CA certificate and private key in order to forge TLS/SSL certificates on behalf on the remote HTTPs server.
If none are specified, the forward proxy will throw a warning at startup and refuse to proxy HTTPs requests.

The following commands generate both, valid for a year, without any passphrase (otherwise, the proxy would require us to type it).

```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ca.key -out ca.crt
```

The certificate then has to be distributed and trusted by every clients that goes through the forward proxy.
The private key must remain secret.

The CA certificate and private key should be specified using the `ca_key_file` and `ca_crt_file` parameters in the [Signer configuration](#signer-config).

## Generate reverse proxy's key pair

To confirm reverse proxy's authenticity and to guarantee confidentiality and integrity of the exchanged data, the reverse proxy can terminate TLS/SSL using a public/private key pair.
The key pair could either be provided by a trusted certificate authority or self-signed using the commands below:

```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout reverseProxy.key -out reverseProxy.crt
```

Note that to the question `Common Name (e.g. server FQDN or YOUR name)`, you must write the host that you expect to use for the reverse proxy.

The key pair should be specified using the `key_file` and `crt_file` parameters in the [Verifier configuration](#verifier-config).
Also, because the key pair is self-signed, the certificate must be trusted by the forward proxy. This can be done by trusting the certificate system-wide or by specifying it using the `trusted_certificates` list parameter in the [Signer configuration](#signer-config).
