Neptulon
========

[![Build Status](https://travis-ci.org/nbusy/neptulon.svg?branch=master)](https://travis-ci.org/nbusy/neptulon) [![GoDoc](https://godoc.org/github.com/nbusy/neptulon?status.svg)](https://godoc.org/github.com/nbusy/neptulon)

Neptulon is a socket framework with middleware support. Framework core is built on listener and context objects. Each message on each connection creates a context which is then passed on to the registered middleware for handling.

Framework core is a small ~1000 SLOC codebase which makes it easy to fork, specialize, and maintain for specific purposes, if you need to.

Example
-------

ToDo

License
-------

[MIT](LICENSE)