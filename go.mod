module github.com/bartdeboer/script.v2

go 1.18

// replace github.com/bartdeboer/pipeline => ../pipeline

require (
	github.com/bartdeboer/pipeline v0.0.3
	github.com/google/go-cmp v0.5.9
	github.com/itchyny/gojq v0.12.13
	github.com/rogpeppe/go-internal v1.11.0
	mvdan.cc/sh/v3 v3.7.0
)

require (
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
)
