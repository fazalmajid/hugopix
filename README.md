# Hugopix

## Introduction

Hugopix is a simple tool to create image galleries for use with the
[Hugo](https://gohugo.io/) static site generator.

The assumption is that you use Lightroom or some similar Digital Asset
Management (DAM) software to manage your photos, and want to do as little data
entry as possible outside the DAM.

## Building from source

You need [Go](https://golang.org) as a prerequisite (but then you need it for
Hugo as well).

Git clone or extract the source code from a tarball, then run `make`

## Usage

`hugopix` assumes you store your photos under `content/galleries` in your Hugo
installation (by default `~/hugo/galleries`)
