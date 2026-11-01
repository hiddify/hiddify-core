go mod tidy
TAGS=$(make -s --no-print-directory print-CLI_TAGS)
# TAGS=with_dhcp,with_low_memory,with_conntrack
go run --tags $TAGS ./cmd/main  $@