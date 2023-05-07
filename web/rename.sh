#!/bin/bash

for name in out/**/*.html; do
    mv "$name" "${name%.html}"
done

for name in out/*.html; do
    mv "$name" "${name%.html}"
done