package hcore

// diagnosticsPublicKey is the age X25519 public key (age1…, from `age-keygen`) the sensitive part
// of every diagnostics report is encrypted to, so only the holder of its private key opens the
// config, the logs and the crashes. Set it here, or at build time:
//
//	-ldflags "-X github.com/hiddify/hiddify-core/v2/hcore.diagnosticsPublicKey=age1…"
//
// Empty, the sensitive part is left in the clear. Not an age key, no report is made at all
// rather than one in the clear.
var diagnosticsPublicKey = ""
