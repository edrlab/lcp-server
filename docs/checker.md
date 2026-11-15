---
layout: default
title: LCP Checker
nav_order: 6
---

# Install

## Quick install, test mode
A basic install takes one line:

```bash
// Fetch from the remote repo, test mode:
go install github.com/edrlab/lcp-server/cmd/lcpchecker@latest
```

## Install after cloning, production mode
If you have received the LCP Production Package, a complete install from the cloned lcp-server repository is obtained using:

``bash
go build -tags "PLCP" -o $GOPATH/bin ./cmd/lcpchecker
```

# Usage
Usage: lcpchecker \[-passphrase] \[-level] \[-verbose] filepath

with: 

  -level: test level
    1:  check the license
    2: check the status document
    3: check the fresh license
    4: check that the license can be registered, renewed, returned

  -passphrase:  user passphrase. If not indicated, the passphrase is not tested.

  -verbose: if set, display info messages; if not set, display only warnings and errors.
 
 filepath: file path of the license to be tested. 
