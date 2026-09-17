package phone_test

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/sms"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

var codePattern = regexp.MustCompile(`code is: ([0-9]{6})`)

func TestPhoneOTP(t *testing.T) {
	t.Parallel()

	integration.Test(t, &config.Config{
		SMSProvider: "mock",
	}, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sender sms.Sender,
	) {
		mock := sender.(*sms.Mock)

		lc.Append(fx.StartHook(func() {
			t.Run("code_cannot_be_reused", func(t *testing.T) {
				r := require.New(t)

				handle := "phone-" + xid.New().String()
				phoneNumber := "+1555" + xid.New().String()[13:]

				request, err := cl.PhoneRequestCodeWithResponse(root, nil, openapi.PhoneRequestCodeProps{
					Identifier:  handle,
					PhoneNumber: phoneNumber,
				})
				r.NoError(err)
				r.Equal(http.StatusOK, request.StatusCode())

				msg := mock.GetLast()
				r.Equal(phoneNumber, msg.Phone)
				code := codePattern.FindStringSubmatch(msg.Message)[1]

				first, err := cl.PhoneSubmitCodeWithResponse(root, handle, openapi.PhoneSubmitCodeProps{Code: code})
				r.NoError(err)
				r.Equal(http.StatusOK, first.StatusCode())

				// B6: the same code must not work a second time.
				second, err := cl.PhoneSubmitCodeWithResponse(root, handle, openapi.PhoneSubmitCodeProps{Code: code})
				r.NoError(err)
				r.Equal(http.StatusForbidden, second.StatusCode())
			})

			t.Run("locks_out_after_too_many_wrong_attempts", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				handle := "phone-" + xid.New().String()
				phoneNumber := "+1555" + xid.New().String()[13:]

				request, err := cl.PhoneRequestCodeWithResponse(root, nil, openapi.PhoneRequestCodeProps{
					Identifier:  handle,
					PhoneNumber: phoneNumber,
				})
				r.NoError(err)
				r.Equal(http.StatusOK, request.StatusCode())

				msg := mock.GetLast()
				code := codePattern.FindStringSubmatch(msg.Message)[1]

				for i := 0; i < 5; i++ {
					resp, err := cl.PhoneSubmitCodeWithResponse(root, handle, openapi.PhoneSubmitCodeProps{Code: "000000"})
					r.NoError(err)
					a.Equal(http.StatusForbidden, resp.StatusCode())
				}

				// Even the correct code is rejected once the attempt budget
				// for it is spent.
				resp, err := cl.PhoneSubmitCodeWithResponse(root, handle, openapi.PhoneSubmitCodeProps{Code: code})
				r.NoError(err)
				r.Equal(http.StatusForbidden, resp.StatusCode())
			})
		}))
	}))
}
