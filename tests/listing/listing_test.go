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

			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Market: &openapi.MarketServiceSettings{Enabled: opt.New(true).Ptr()},
				},
			}, adminSession))(t, http.StatusOK)

			guitar := openapi.ListingInitialProps{
				Title:       "Akustikgitarre Yamaha",
				Description: opt.New("Kaum gespielt, mit Tasche.").Ptr(),
				Price:       12000,
				PriceType:   openapi.ListingPriceTypeNegotiable,
				Category:    "instruments",
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

				ok := tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "0151 123 456 78"}, sellerSession))(t, http.StatusOK)
				r.Equal("+4915112345678", *ok.JSON200.Phone)

				get := tests.AssertRequest(cl.AccountPhoneGetWithResponse(root, sellerSession))(t, http.StatusOK)
				r.Equal("+4915112345678", *get.JSON200.Phone)
			})

			t.Run("phone_not_in_public_profile", func(t *testing.T) {
				r := require.New(t)

				prof := tests.AssertRequest(cl.ProfileGetWithResponse(root, seller.Handle, buyerSession))(t, http.StatusOK)
				r.NotContains(string(prof.Body), "15112345678")
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
					Category:  "books",
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
				a.Equal([]string{"Physiologie Lehrbuch"}, ids(openapi.ListingListParams{Category: opt.New[openapi.ListingCategoryQuery]("books").Ptr()}))
				a.Equal([]string{"Akustikgitarre Yamaha"}, ids(openapi.ListingListParams{PriceMin: opt.New("5000").Ptr()}))
				a.Equal([]string{"Physiologie Lehrbuch"}, ids(openapi.ListingListParams{PriceMax: opt.New("100").Ptr()}))
				a.Equal([]string{"Physiologie Lehrbuch", "Akustikgitarre Yamaha"}, ids(openapi.ListingListParams{Sort: opt.New(openapi.ListingListParamsSortPriceAsc).Ptr()}))

				bounds := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{PriceMin: opt.New("5000").Ptr()}, strangerSession))(t, http.StatusOK)
				a.Equal(0, bounds.JSON200.PriceMin)
				a.Equal(12000, bounds.JSON200.PriceMax)

				books := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{Category: opt.New[openapi.ListingCategoryQuery]("books").Ptr()}, strangerSession))(t, http.StatusOK)
				a.Equal(0, books.JSON200.PriceMin)
				a.Equal(0, books.JSON200.PriceMax)
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
				a.Equal("+4915112345678", contact.JSON200.Phone)

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

				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				a.Equal(14, *info.JSON200.MarketplaceListingDurationDays)

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

func TestListingsBetaVisibility(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)
			a := assert.New(t)

			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
			r.NotNil(info.JSON200.MarketplaceEnabled)
			a.False(*info.JSON200.MarketplaceEnabled)

			list, err := cl.ListingListWithResponse(root, &openapi.ListingListParams{}, memberSession)
			tests.Status(t, err, list, http.StatusNotFound)

			tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "+436641234567"}, adminSession))(t, http.StatusOK)
			created := tests.AssertRequest(cl.ListingCreateWithResponse(root, openapi.ListingInitialProps{
				Title:     "Beta Testinserat",
				Price:     100,
				PriceType: openapi.ListingPriceTypeFixed,
				Category:  "other",
				Condition: openapi.ListingConditionNew,
			}, adminSession))(t, http.StatusOK)

			get, err := cl.ListingGetWithResponse(root, created.JSON200.Slug, memberSession)
			tests.Status(t, err, get, http.StatusNotFound)

			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Market: &openapi.MarketServiceSettings{Enabled: opt.New(true).Ptr()},
				},
			}, adminSession))(t, http.StatusOK)

			info = tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
			a.True(*info.JSON200.MarketplaceEnabled)

			tests.AssertRequest(cl.ListingGetWithResponse(root, created.JSON200.Slug, memberSession))(t, http.StatusOK)

			for _, enabled := range []bool{false, true} {
				tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Market: &openapi.MarketServiceSettings{Enabled: opt.New(enabled).Ptr()},
					},
				}, adminSession))(t, http.StatusOK)

				info = tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				a.Equal(enabled, *info.JSON200.MarketplaceEnabled)
			}

			settings := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, adminSession))(t, http.StatusOK)
			a.True(*settings.JSON200.Services.Market.Enabled)
			a.Equal(90, *settings.JSON200.Services.Market.ListingDurationDays)
			a.Equal(90, *info.JSON200.MarketplaceListingDurationDays)
		}))
	}))
}

func TestListingsAdminCategories(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)
			a := assert.New(t)

			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
			r.NotNil(info.JSON200.MarketplaceCategories)
			a.Len(*info.JSON200.MarketplaceCategories, len(settings.DefaultMarketCategories))

			defaults := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, adminSession))(t, http.StatusOK)
			r.NotNil(defaults.JSON200.Services.Market.Categories)
			a.Equal(*info.JSON200.MarketplaceCategories, *defaults.JSON200.Services.Market.Categories)

			categories := openapi.MarketCategoryList{
				{Id: "instruments", Name: "Instrumente"},
				{Id: "skripten", Name: "Skripten"},
			}
			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Market: &openapi.MarketServiceSettings{Enabled: opt.New(true).Ptr(), Categories: &categories},
				},
			}, adminSession))(t, http.StatusOK)

			info = tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
			a.Equal(categories, *info.JSON200.MarketplaceCategories)

			got := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, adminSession))(t, http.StatusOK)
			a.Equal(categories, *got.JSON200.Services.Market.Categories)

			tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "+436641234568"}, memberSession))(t, http.StatusOK)

			props := openapi.ListingInitialProps{
				Title:     "Histologie Skript",
				Price:     500,
				PriceType: openapi.ListingPriceTypeFixed,
				Category:  "skripten",
				Condition: openapi.ListingConditionUsed,
			}
			created := tests.AssertRequest(cl.ListingCreateWithResponse(root, props, memberSession))(t, http.StatusOK)
			a.Equal("skripten", created.JSON200.Category)

			filtered := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{Category: opt.New[openapi.ListingCategoryQuery]("skripten").Ptr()}, memberSession))(t, http.StatusOK)
			a.Len(filtered.JSON200.Listings, 1)

			props.Category = "books"
			unknown, err := cl.ListingCreateWithResponse(root, props, memberSession)
			tests.Status(t, err, unknown, http.StatusBadRequest)

			renamed := openapi.MarketCategoryList{{Id: "instruments", Name: "Instrumente"}}
			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Market: &openapi.MarketServiceSettings{Categories: &renamed},
				},
			}, adminSession))(t, http.StatusOK)

			kept := tests.AssertRequest(cl.ListingGetWithResponse(root, created.JSON200.Slug, memberSession))(t, http.StatusOK)
			a.Equal("skripten", kept.JSON200.Category)

			tests.AssertRequest(cl.ListingUpdateWithResponse(root, created.JSON200.Slug, openapi.ListingMutableProps{Title: opt.New("Histologie Skript v2").Ptr(), Category: opt.New[openapi.ListingCategory]("skripten").Ptr()}, memberSession))(t, http.StatusOK)

			for _, bad := range []openapi.MarketCategoryList{
				{},
				{{Id: "a", Name: "A"}, {Id: "a", Name: "B"}},
				{{Id: "Bad Id", Name: "A"}},
			} {
				resp, err := cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Market: &openapi.MarketServiceSettings{Categories: &bad},
					},
				}, adminSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			}
		}))
	}))
}

func TestListingsModerator(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		repo *listing_repo.Repository,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)
			a := assert.New(t)

			sellerCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			sellerSession := sh.WithSession(sellerCtx)
			modCtx, mod := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			modSession := sh.WithSession(modCtx)
			strangerCtx, _ := e2e.WithAccount(root, aw, seed.Account_005_Þórr)
			strangerSession := sh.WithSession(strangerCtx)
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Market: &openapi.MarketServiceSettings{Enabled: opt.New(true).Ptr()},
				},
			}, adminSession))(t, http.StatusOK)

			role := tests.AssertRequest(cl.RoleCreateWithResponse(root, openapi.RoleCreateJSONRequestBody{
				Name:        "flohmarkt-mod",
				Colour:      "blue",
				Permissions: openapi.PermissionList{openapi.MANAGELISTINGS},
			}, adminSession))(t, http.StatusOK)
			tests.AssertRequest(cl.AccountAddRoleWithResponse(root, mod.Handle, role.JSON200.Id, adminSession))(t, http.StatusOK)

			tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "+436641234569"}, sellerSession))(t, http.StatusOK)
			tests.AssertRequest(cl.AccountPhoneSetWithResponse(root, openapi.AccountPhoneSetProps{Phone: "+436641234570"}, strangerSession))(t, http.StatusOK)

			created := tests.AssertRequest(cl.ListingCreateWithResponse(root, openapi.ListingInitialProps{
				Title:     "Anatomie Atlas",
				Price:     3000,
				PriceType: openapi.ListingPriceTypeFixed,
				Category:  "books",
				Condition: openapi.ListingConditionUsed,
			}, sellerSession))(t, http.StatusOK).JSON200

			tests.AssertRequest(cl.ListingOfferSubmitWithResponse(root, created.Slug, openapi.ListingOfferProps{Amount: 2500}, strangerSession))(t, http.StatusOK)

			t.Run("can_manage_flag", func(t *testing.T) {
				asMod := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, modSession))(t, http.StatusOK)
				a.True(asMod.JSON200.CanManage)
				a.False(asMod.JSON200.IsOwn)
				r.NotNil(asMod.JSON200.OfferCount)
				a.Equal(1, *asMod.JSON200.OfferCount)

				asStranger := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, strangerSession))(t, http.StatusOK)
				a.False(asStranger.JSON200.CanManage)
				a.Nil(asStranger.JSON200.OfferCount)

				asSeller := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, sellerSession))(t, http.StatusOK)
				a.True(asSeller.JSON200.CanManage)
			})

			t.Run("cannot_update", func(t *testing.T) {
				edit, err := cl.ListingUpdateWithResponse(root, created.Slug, openapi.ListingMutableProps{
					Title: opt.New("Anatomie Atlas (moderiert)").Ptr(),
				}, modSession)
				tests.Status(t, err, edit, http.StatusForbidden)

				status, err := cl.ListingUpdateWithResponse(root, created.Slug, openapi.ListingMutableProps{
					Status: opt.New(openapi.ListingStatusReserved).Ptr(),
				}, modSession)
				tests.Status(t, err, status, http.StatusForbidden)

				adminStatus, err := cl.ListingUpdateWithResponse(root, created.Slug, openapi.ListingMutableProps{
					Status: opt.New(openapi.ListingStatusSold).Ptr(),
				}, adminSession)
				tests.Status(t, err, adminStatus, http.StatusForbidden)

				get := tests.AssertRequest(cl.ListingGetWithResponse(root, created.Slug, sellerSession))(t, http.StatusOK)
				a.Equal("Anatomie Atlas", get.JSON200.Title)
				a.Equal(openapi.ListingStatusActive, get.JSON200.Status)
			})

			t.Run("offers", func(t *testing.T) {
				denied, err := cl.ListingOfferListWithResponse(root, created.Slug, strangerSession)
				tests.Status(t, err, denied, http.StatusForbidden)

				offers := tests.AssertRequest(cl.ListingOfferListWithResponse(root, created.Slug, modSession))(t, http.StatusOK)
				r.Len(offers.JSON200.Offers, 1)
				a.Equal(2500, offers.JSON200.Offers[0].Amount)
				r.NotNil(offers.JSON200.Offers[0].BuyerPhone)
			})

			_, err := repo.Update(root, listing.ID(openapi.ParseID(created.Id)), listing_repo.WithExpiresAt(time.Now().Add(-time.Hour)))
			r.NoError(err)

			t.Run("manage_scope", func(t *testing.T) {
				manage := opt.New(openapi.ListingListParamsScopeManage).Ptr()

				denied, err := cl.ListingListWithResponse(root, &openapi.ListingListParams{Scope: manage}, strangerSession)
				tests.Status(t, err, denied, http.StatusForbidden)

				list := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{Scope: manage}, modSession))(t, http.StatusOK)
				r.Len(list.JSON200.Listings, 1)
				a.Equal(created.Id, list.JSON200.Listings[0].Id)
				a.True(list.JSON200.Listings[0].IsExpired)
				a.True(list.JSON200.Listings[0].CanManage)

				all := tests.AssertRequest(cl.ListingListWithResponse(root, &openapi.ListingListParams{}, modSession))(t, http.StatusOK)
				a.Len(all.JSON200.Listings, 0)
			})

			t.Run("renew", func(t *testing.T) {
				denied, err := cl.ListingRenewWithResponse(root, created.Slug, strangerSession)
				tests.Status(t, err, denied, http.StatusForbidden)

				renewed := tests.AssertRequest(cl.ListingRenewWithResponse(root, created.Slug, modSession))(t, http.StatusOK)
				a.False(renewed.JSON200.IsExpired)
			})

			t.Run("delete", func(t *testing.T) {
				tests.AssertRequest(cl.ListingDeleteWithResponse(root, created.Slug, modSession))(t, http.StatusOK)

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
