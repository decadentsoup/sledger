#!/bin/sh -e
cd "$(dirname "$0")/.." && docker build --tag sledger:test .
