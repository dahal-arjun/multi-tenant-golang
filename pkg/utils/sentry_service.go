package utils

import sentry "github.com/getsentry/sentry-go"

type SentryService interface {
	CaptureException(err error)
}

type DefaultSentryService struct{}

func (s *DefaultSentryService) CaptureException(err error) {
	sentry.CaptureException(err)
}

var CurrentSentryService SentryService = &DefaultSentryService{}
