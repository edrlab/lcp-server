---
layout: default
title: Configuration
nav_order: 2
---

# Configuration 

The configuration of the server is possible using a configuration file and/or environment variables. It is possible to mix both sets;  environment variables are expressly recommended for confidential information. 

The configuration file is formatted as yaml, and can be located in any folder directly accessible from the application. The configuration file is found by the application via an environment variable named `LCPSERVER_CONFIG`. Its value must be a file path.

Configuration properties are expressed in snake case in the configuration file, and all caps with no underscore, prefixed by `LCPSERVER` plus their sub-section name, when expressed as environment variables. 
As an example, the `port` configuration property is mapped to the `LCPSERVER_PORT` environment variable, `public_base_url` becomes `LCPSERVER_PUBLICBASEURL`, and the `username` property of the `access` section becomes `LCPSERVER_ACCESS_USERNAME`.

For now, follow this example.

```yaml
# log level, can be "debug", "info", "warn", "error"
log_level: "debug"

# the public url of the server (used for setting links in the status document)
public_base_url: "https://lcp.edrlab.org"
# the port used by the server
port: 8989
# data source name of access to the chosen database
dsn: "sqlite3://file::memory:?cache=shared"

# username / password allowing access to the server API via http basic authentication
# for security reasons, it is much better to express these as environment variables (see the documentation)
# and docker secrets (https://docs.docker.com/compose/how-tos/use-secrets/)
access:
  username: "login"
  password: "password"

license:
  # provider identifier, as a url, set in every license
  provider: "http://edrlab.org"
  # LCP profile identifier, set in every license, can be overridden per license
  profile: "http://readium.org/lcp/basic-profile"
  # link to a hint page, can be templated using {license_id} as parameter
  hint_link: "https://lcp.edrlab.org/help/{license_id}"

status:
  # url of a fresh license, served via a License Gateway 
  # must be templated using {license_id} as parameter
  fresh_license_link: "https://lcp.edrlab.org/freshlicense/{license_id}"
  # max number of days of extension of a license. 
  # if not set ot set to zero, there will be not renew/return links in status documents
  renew_max_days: 40
  # allow renew on expired licenses; false if not set.
  allow_renew_on_expired_licenses: true
  # default number of days of extension of a license, see renew; if not set, the default value is 7. Can be overridden in the renew command
  renew_default_days: 7
  # renew URL optionally managed by the provider, which then takes care of calling the license status server.
  # standard behavior if not set. 
  # must be templated using {license_id} as parameter
  renew_link: "http://lcp.edrlab.org/custom/renew/{license_id}"

dashboard:
  # configurable threshold for licenses with excessive sharing (default is 6)
  excessive_sharing_threshold: 10
  # optional limit to last 12 months (default is false)
  limit_to_last_12_months: true

# path to the X509 certificate and private key used for signing licenses
certificate:
  cert:       "/config/cert-edrlab-test.pem"
  private_key: "/config/privkey-edrlab-test.pem"
```

The EDRLab LCP test certificate and private key are provided in the source-code project, in the /test/cert folder. They are only useful during a testing phase, and will be replaced by a production certificate provided by EDRLab when the system is ready for production.  