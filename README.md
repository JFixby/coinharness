# Coin harness
Satoshi regression testing framework

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

Provides a general test framework for crafting and executing integration tests
a coin node instance via the `RPC` interface. Each instance of an active harness
comes equipped with a wallet capable of properly syncing to the generated chain,
creating new addresses, and crafting fully signed transactions paying to an
arbitrary set of outputs. 

Harness fully encapsulates an active node process to provide a unified
platform for creating rpc driven integration tests involving coin-node. The
active node will typically be run in simnet mode in order to allow for
easy generation of test blockchains.  The active node process is fully
managed by Harness, which handles the necessary initialization, and teardown
of the process along with any temporary directories created as a result.
Multiple Harness instances may be run concurrently, in order to allow for
testing complex scenarios involving multiple nodes. The harness also
includes an wallet to streamline various classes of tests.

This package was designed specifically to act as an RPC testing harness for
`Decred` and `Bitcoin`. However, the constructs presented are general enough to be
adapted to any project wishing to programmatically drive a node instance of its
systems/integration tests.

## Applications

 - [Decred regressions testing](https://github.com/JFixby/dcrregtest)

 - [Bitcoin regressions testing](https://github.com/JFixby/btcregtest)

## License

Licensed under the [copyfree](http://copyfree.org) ISC License.


