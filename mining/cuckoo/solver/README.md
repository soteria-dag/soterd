external cuckoo solvers
======

This directory contains a copy of [John Tromp's cuckoo solvers](https://github.com/tromp/cuckoo), in order to simplify its build and use with soterd.

The solver and verifier code has been modified to:
* Allow specifying a file containing block header data to perform Proof-of-Work or verification for.
* Fix an error in the verification code, that would prematurely stop reading header data if it encountered a null-byte before the end of the header.

Instructions for building and using the external solvers is in the [External Cuckoo Solver](../../../docs/external_cuckoo_solver.md) document.  