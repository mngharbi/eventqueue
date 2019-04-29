# EventQueue  ![Travis CI Build Status](https://api.travis-ci.org/mngharbi/eventqueue.svg?branch=master) [![Go Report Card](https://goreportcard.com/badge/gojp/goreportcard)](https://goreportcard.com/report/mngharbi/eventqueue) [![Coverage](https://codecov.io/gh/mngharbi/eventqueue/branch/master/graph/badge.svg)](https://codecov.io/gh/mngharbi/eventqueue) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/mngharbi/eventqueue/master/LICENSE)

## Overview

EventQueue implements a fast goroutine safe pubsub for one topic that maintains publish order while being non-blocking on publish.

It offers 5 simple API calls on an eventqueue object:
- Publish
- Subscribe
- Unsubscribe
- Done: closes the queue
- Wait: blocks until the queue is closed and all subscribers are done reading.
- WaitUntilEmpty: blocks until the queue is empty
