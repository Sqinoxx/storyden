package listing_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/listing"
	"github.com/Southclaws/storyden/app/resources/listing/listing_repo"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/listing/listing_expiry"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestListings(t *testing.T) {
	t.Parallel()

	integration.Test(t, &config.Config{OCREnabled: false, OCRBackfillEnabled: false}, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		repo *listing_repo.Repository,
		expiry *listing_expiry.Expiry,
	) {
		lc.Append(fx.StartHook(func() {
			sellerCtx, seller := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			sellerSession := sh.WithSession(sellerCtx)
			buyerCtx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			buyerSession := sh.WithSession(buyerCtx)
			strangerCtx, _ := e2e.WithAccount(root, aw, seed.Account_005_Þórr)
			strangerSession := sh.WithSession(strangerCtx)
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			guitar := openapi.ListingInitialProps{
				Title:       "Akustikgitarre Yamaha",
				Description: opt.New("Kaum gespielt, mit Tasche.").Ptr(),
				Price:       12000,
				PriceType:   openapi.ListingPriceTypeNegotiable,
				Category:    openapi.ListingCategoryInstruments,
				Condition:   openapi.ListingConditionLikeNew,
				Location:    opt.New("Wien").Ptr(),
			}

			t.Run("guest_cannot_list", func(t *testing.T) {
				resp, err := cl.ListingListWithResponse(root, &openapi.ListingListParams{})
				tests.Status(t, err, resp, http.StatusUnauthorized)
			})

			t.Run("create_requires_phone", func(t *testing.T) {
				resp, err := cl.ListingCreateWithResponse(root, guitar, sellerSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("phone_validation", func(t *testing.T) {
				r := require.New(t)

				bad, err := cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "abc123"}, sellerSession)
				tests.Status(t, err, bad, http.StatusBadRequest)

				ok := tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "0664 123 45 67"}, sellerSession))(t, http.StatusOK)
				r.Equal("+436641234567", *ok.JSON200.Phone)

				get := tests.AssertRequest(cl.AccountPhoneGetWithResponse(root, sellerSession))(t, http.StatusOK)
				r.Equal("+436641234567", *get.JSON200.Phone)
			})

			t.Run("phone_not_in_public_profile", func(t *testing.T) {
				r := require.New(t)

				prof := tests.AssertRequest(cl.ProfileGetWithResponse(root, seller.Handle, buyerSession))(t, http.StatusOK)
				r.NotContains(string(prof.Body), "6641234567")
			})

			image1 := uploadPNG(t, root, cl, sellerSession, "a.png")
			image2 := uploadPNG(t, root, cl, sellerSession, "b.png")

			var created *openapi.Listing

			t.Run("create_with_images", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				body := guitar
				body.AssetIds = &[]string{image2.Id, image1.Id}

				resp := tests.AssertRequest(cl.ListingCreateWithResponse(root, body, sellerSession))(t, http.StatusOK)
				created = resp.JSON200

				a.Equal(guitar.Title, created.Title)
				a.Equal(12000, created.Price)
				a.True(created.IsOwn)
				a.False(created.IsExpired)
				r.Len(created.Assets, 2)
				a.Equal(image2.Id, created.Assets[0].Id)
				a.Equal(image1.Id, created.Assets[1].Id)
				a.WithinDuration(time.Now().Add(settings.DefaultListingDurationDays*24*time.Hour), created.ExpiresAt, time.Minute)
				r.NotNil(created.OfferCount)
				a.Equal(0, *created.OfferCount)
			})

			t.Run("search", func(t *testing.T) {
				a := assert.New(t)

				tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "+4915112345678"}, buyerSession))(t, http.StatusOK)
				tests.AssertRequest(cl.ListingCreateWithResponse(root, openapi.ListingInitialProps{
					Title:     "Physiologie Lehrbuch",
					Price:     0,
					PriceType: openapi.ListingPriceTypeFree,
					Category:  openapi.ListingCategoryBooks,
					Condition: openapi.ListingConditionUsed,
				}, buyerSession))(t, http.StatusOK)

				ids := func(p openapi.ListingListParams) []string {
					resp := tests.AssertRequest(cl.ListingListWithResponse(root, &p, strangerSession))(t, http.StatusOK)
					out := []string{}
					for _, l := range resp.JSON200.Listings {
						out = append(out, l.Title)
					}
					return out
				}

				a.ElementsMatch([]string{"Akustikgitarre Yamaha", "Physiologie Lehrbuch"}, ids(openapi.ListingListParams{}))
				a.Equal([]string{"Akustikgitarre Yamaha"}, ids(openapi.ListingListParams{Q: opt.New("gitarre").Ptr()}))
				a.Equal([]string{"Akustikgitarre Yamaha"}, ids(openapi.ListingListParams{Q: opt.New("wien").Ptr()}))
				a.Equal([]string{"Physiologie Lehrbuch"}, ids(openapi.ListingListParams{Category: opt.New(openapi.ListingCategoryBooks).Ptr()}))
				a.Equal([]string{"Akustikgitarre Yamaha"}, ids(openapi.ListingListParams{PriceMin: opt.New("5000").Ptr()}))
				a.Equal([]string{"Physiologie Lehrbuch"}, ids(openapi.ListingListParams{PriceMax: opt.New("100").Ptr()}))
				a.Equal([]string{"Physiologie Lehrbuch", "Akustikgitarre Yamaha"}, ids(openapi.ListingListParams{Sort: opt.New(openapi.ListingListParamsSortPriceAsc).Ptr()}))
			})

			t.Run("contact_hidden_without_offer", func(t *testing.T) {
				resp, err := cl.ListingContactGetWithResponse(root, created.Slug, buyerSession)
				tests.Status(t, err, resp, http.StatusForbidden)

				offers, err := cl.ListingOfferListWithResponse(root, created.Slug, buyerSession)
				tests.Status(t, err, offers, http.StatusForbidden)
			})

			t.Run("cannot_offer_on_own_listing", func(t *testing.T) {
				resp, err := cl.ListingOfferSubmitWithResponse(root, created.Slug, openapi.ListingOfferProps{Amount: 100}, sellerSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("offer_requires_phone", func(t *testing.T) {
				resp, err := cl.ListingOfferSubmitWithResponse(root, created.Slug, openapi.ListingOfferProps{Amount: 100}, strangerSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("offer_reveals_phones_both_ways", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				offer := tests.AssertRequest(cl.ListingOfferSubmitWithResponse(root, created.Slug, openapi.ListingOfferProps{
					Amount:  10000,
					Message: opt.New("Würde sie am Wochenende abholen.").Ptr(),
				}, buyerSession))(t, http.StatusOK)
				a.Equal(10000, offer.JSON200.Amount)
				a.Nil(offer.JSON200.BuyerPhone)

				contact := tests.AssertRequest(cl.ListingContactGetWithResponse(root, created.Slug, buyerSession))(t, http.StatusOK)
				a.Equal("+436641234567", contact.JSON200.Phone)

				get := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, buyerSession))(t, http.StatusOK)
				a.True(get.JSON200.HasOffered)
				a.Nil(get.JSON200.OfferCount)

				tests.AssertRequest(cl.ListingOfferSubmitWithResponse(root, created.Slug, openapi.ListingOfferProps{Amount: 11000}, buyerSession))(t, http.StatusOK)

				offers := tests.AssertRequest(cl.ListingOfferListWithResponse(root, created.Slug, sellerSession))(t, http.StatusOK)
				r.Len(offers.JSON200.Offers, 1)
				a.Equal(11000, offers.JSON200.Offers[0].Amount)
				r.NotNil(offers.JSON200.Offers[0].BuyerPhone)
				a.Equal("+4915112345678", *offers.JSON200.Offers[0].BuyerPhone)

				own := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, sellerSession))(t, http.StatusOK)
				r.NotNil(own.JSON200.OfferCount)
				a.Equal(1, *own.JSON200.OfferCount)
			})

			t.Run("favourites", func(t *testing.T) {
				r := require.New(t)

				tests.AssertRequest(cl.ListingFavouriteAddWithResponse(root, created.Slug, strangerSession))(t, http.StatusOK)
				tests.AssertRequest(cl.ListingFavouriteAddWithResponse(root, created.Slug, strangerSession))(t, http.StatusOK)

				favs := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{Scope: opt.New(openapi.ListingListParamsScopeFavourites).Ptr()}, strangerSession))(t, http.StatusOK)
				r.Len(favs.JSON200.Listings, 1)
				r.True(favs.JSON200.Listings[0].IsFavourite)

				tests.AssertRequest(cl.ListingFavouriteRemoveWithResponse(root, created.Slug, strangerSession))(t, http.StatusOK)

				favs = tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{Scope: opt.New(openapi.ListingListParamsScopeFavourites).Ptr()}, strangerSession))(t, http.StatusOK)
				r.Len(favs.JSON200.Listings, 0)
			})

			t.Run("update_permissions", func(t *testing.T) {
				a := assert.New(t)

				resp, err := cl.ListingUpdateWithResponse(root, created.Slug, openapi.ListingMutableProps{Title: opt.New("Gestohlen").Ptr()}, strangerSession)
				tests.Status(t, err, resp, http.StatusForbidden)

				upd := tests.AssertRequest(cl.ListingUpdateWithResponse(root, created.Slug, openapi.ListingMutableProps{
					Status: opt.New(openapi.ListingStatusReserved).Ptr(),
					Price:  opt.New(11000).Ptr(),
				}, sellerSession))(t, http.StatusOK)
				a.Equal(openapi.ListingStatusReserved, upd.JSON200.Status)
				a.Equal(11000, upd.JSON200.Price)
			})

			t.Run("renew_too_early", func(t *testing.T) {
				resp, err := cl.ListingRenewWithResponse(root, created.Slug, sellerSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("expiry_notification_sweep", func(t *testing.T) {
				a := assert.New(t)

				_, err := repo.Update(root, listing.ID(openapi.ParseID(created.Id)), listing_repo.WithExpiresAt(time.Now().Add(48*time.Hour)))
				require.NoError(t, err)

				a.GreaterOrEqual(expiry.Sweep(root, time.Now()), 1)
				a.Equal(0, expiry.Sweep(root, time.Now()))
			})

			t.Run("expired_hidden_then_renewed", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				_, err := repo.Update(root, listing.ID(openapi.ParseID(created.Id)), listing_repo.WithExpiresAt(time.Now().Add(-time.Hour)))
				r.NoError(err)

				list := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{}, strangerSession))(t, http.StatusOK)
				for _, l := range list.JSON200.Listings {
					a.NotEqual(created.Id, l.Id)
				}

				hidden, err := cl.ListingGetWithResponse(root, created.Slug, strangerSession)
				tests.Status(t, err, hidden, http.StatusNotFound)

				stillVisibleToBuyer := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, buyerSession))(t, http.StatusOK)
				a.True(stillVisibleToBuyer.JSON200.IsExpired)

				offer, err := cl.ListingOfferSubmitWithResponse(root, created.Slug, openapi.ListingOfferProps{Amount: 1}, buyerSession)
				tests.Status(t, err, offer, http.StatusBadRequest)

				mine := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{Scope: opt.New(openapi.ListingListParamsScopeMine).Ptr()}, sellerSession))(t, http.StatusOK)
				r.Len(mine.JSON200.Listings, 1)
				a.True(mine.JSON200.Listings[0].IsExpired)

				renewed := tests.AssertRequest(cl.ListingRenewWithResponse(root, created.Slug, sellerSession))(t, http.StatusOK)
				a.False(renewed.JSON200.IsExpired)

				tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, strangerSession))(t, http.StatusOK)
			})

			t.Run("admin_duration_setting", func(t *testing.T) {
				a := assert.New(t)

				tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Market: &openapi.MarketServiceSettings{ListingDurationDays: opt.New(14).Ptr()},
					},
				}, adminSession))(t, http.StatusOK)

				get := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, adminSession))(t, http.StatusOK)
				a.Equal(14, *get.JSON200.Services.Market.ListingDurationDays)

				resp := tests.AssertRequest(cl.ListingCreateWithResponse(root, guitar, sellerSession))(t, http.StatusOK)
				a.WithinDuration(time.Now().Add(14*24*time.Hour), resp.JSON200.ExpiresAt, time.Minute)

				invalid, err := cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Market: &openapi.MarketServiceSettings{ListingDurationDays: opt.New(0).Ptr()},
					},
				}, adminSession)
				tests.Status(t, err, invalid, http.StatusBadRequest)
			})

			t.Run("delete", func(t *testing.T) {
				stranger, err := cl.ListingDeleteWithResponse(root, created.Slug, strangerSession)
				tests.Status(t, err, stranger, http.StatusForbidden)

				tests.AssertRequest(cl.ListingDeleteWithResponse(root, created.Slug, adminSession))(t, http.StatusOK)

				gone, err := cl.ListingGetWithResponse(root, created.Slug, sellerSession)
				tests.Status(t, err, gone, http.StatusNotFound)
			})
		}))
	}))
}

func uploadPNG(t *testing.T, ctx context.Context, cl *openapi.ClientWithResponses, session openapi.RequestEditorFn, filename string) openapi.Asset {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	data := buf.Bytes()

	resp, err := cl.AssetUploadWithBodyWithResponse(
		ctx,
		&openapi.AssetUploadParams{
			ContentLength: int64(len(data)),
			Filename:      &filename,
		},
		"image/png",
		bytes.NewReader(data),
		session,
	)
	tests.Ok(t, err, resp)

	return *resp.JSON200
}
