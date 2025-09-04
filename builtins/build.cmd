rsrc -manifest app.manifest -ico=icon/icon.ico -o resource_windows.syso
peg -switch -inline dice/roll.peg
set GORELEASER_CURRENT_TAG=1.0.1
goreleaser release --snapshot --rm-dist
