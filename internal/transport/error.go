package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/url"

	"apitool/internal/model"
)

func transportError(err error) *model.ExecutionError {
	if errors.Is(err, context.Canceled) {
		return safeError(model.StageTransport, model.CategoryCanceled, "Request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return safeError(model.StageTransport, model.CategoryTimeout, "Request timed out")
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return safeError(model.StageTransport, model.CategoryDNS, "DNS lookup failed")
	}
	if isTLSError(err) {
		return safeError(model.StageTransport, model.CategoryTLS, "TLS connection failed")
	}
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		return safeError(model.StageTransport, model.CategoryTimeout, "Request timed out")
	}
	var opError *net.OpError
	if errors.As(err, &opError) {
		return safeError(model.StageTransport, model.CategoryConnection, "Connection failed")
	}
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return safeError(model.StageTransport, model.CategoryConnection, "HTTP request failed")
	}
	return safeError(model.StageTransport, model.CategoryConnection, "HTTP request failed")
}

func isTLSError(err error) bool {
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalidCertificate x509.CertificateInvalidError
	var certificateVerification *tls.CertificateVerificationError
	var recordHeader *tls.RecordHeaderError
	return errors.As(err, &unknownAuthority) || errors.As(err, &hostname) || errors.As(err, &invalidCertificate) || errors.As(err, &certificateVerification) || errors.As(err, &recordHeader)
}

func safeError(stage model.ExecutionStage, category model.ExecutionCategory, message string) *model.ExecutionError {
	return &model.ExecutionError{Stage: stage, Category: category, SafeMessage: message}
}
