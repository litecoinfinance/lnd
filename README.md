## Lightning Network Daemon

[![Build Status](https://img.shields.io/travis/litecoinfinance/lnd.svg)](https://travis-ci.org/litecoinfinance/lnd)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/litecoinfinance/lnd/blob/master/LICENSE)
[![Godoc](https://godoc.org/github.com/litecoinfinance/lnd?status.svg)](https://godoc.org/github.com/litecoinfinance/lnd)

<img src="logo.png">


* [Fast Setup Litecoin Finance ligthning network](https://github.com/litecoinfinance/lnd/blob/master/docs/Fast_Setup.md)


The Lightning Network Daemon (`lnd`) - is a complete implementation of a
[Lightning Network](https://lightning.network) node.  `lnd` has several pluggable back-end
chain services including [`btcd`](https://github.com/litecoinfinance/btcd) (a
full-node), [`bitcoind`](https://github.com/bitcoin/bitcoin), and
[`neutrino`](https://github.com/litecoinfinance/neutrino) (a new experimental light client). The project's codebase uses the
[btcsuite](https://github.com/btcsuite/) set of Bitcoin libraries, and also
exports a large set of isolated re-usable Lightning Network related libraries
within it.  In the current state `lnd` is capable of:
* Creating channels.
* Closing channels.
* Completely managing all channel states (including the exceptional ones!).
* Maintaining a fully authenticated+validated channel graph.
* Performing path finding within the network, passively forwarding incoming payments.
* Sending outgoing [onion-encrypted payments](https://github.com/litecoinfinance/lightning-onion)
through the network.
* Updating advertised fee schedules.
* Automatic channel management ([`autopilot`](https://github.com/litecoinfinance/lnd/tree/master/autopilot)).

## Lightning Network Specification Compliance
`lnd` _fully_ conforms to the [Lightning Network specification
(BOLTs)](https://github.com/lightningnetwork/lightning-rfc). BOLT stands for:
Basis of Lightning Technology. The specifications are currently being drafted
by several groups of implementers based around the world including the
developers of `lnd`. The set of specification documents as well as our
implementation of the specification are still a work-in-progress. With that
said, the current status of `lnd`'s BOLT compliance is:

  - [X] BOLT 1: Base Protocol
  - [X] BOLT 2: Peer Protocol for Channel Management
  - [X] BOLT 3: Bitcoin Transaction and Script Formats
  - [X] BOLT 4: Onion Routing Protocol
  - [X] BOLT 5: Recommendations for On-chain Transaction Handling
  - [X] BOLT 7: P2P Node and Channel Discovery
  - [X] BOLT 8: Encrypted and Authenticated Transport
  - [X] BOLT 9: Assigned Feature Flags
  - [X] BOLT 10: DNS Bootstrap and Assisted Node Location
  - [X] BOLT 11: Invoice Protocol for Lightning Payments


## Installation
  In order to build from source, please see [the installation
  instructions](docs/INSTALL.md).


## Further reading
* [Step-by-step send payment guide with docker](https://github.com/litecoinfinance/lnd/tree/master/docker)
* [Contribution guide](https://github.com/litecoinfinance/lnd/blob/master/docs/code_contribution_guidelines.md)
