#!/bin/bash

cfssl genkey csr.json | cfssljson -bare certificate
