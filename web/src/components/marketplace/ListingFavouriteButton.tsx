"use client";

import { useState } from "react";

import { handle } from "@/api/client";
import {
  listingFavouriteAdd,
  listingFavouriteRemove,
} from "@/api/openapi-client/listings";
import { Listing } from "@/api/openapi-schema";
import { IconButton } from "@/components/ui/icon-button";
import { LikeIcon, LikeSavedIcon } from "@/components/ui/icons/Like";
import { useTranslation } from "@/lib/i18n";

type Props = {
  listing: Listing;
  onChange?: () => void;
};

export function ListingFavouriteButton({ listing, onChange }: Props) {
  const t = useTranslation();
  const [favourite, setFavourite] = useState(listing.is_favourite);

  if (listing.is_own) {
    return null;
  }

  async function handleToggle() {
    const next = !favourite;
    setFavourite(next);
    await handle(
      async () => {
        if (next) {
          await listingFavouriteAdd(listing.slug);
        } else {
          await listingFavouriteRemove(listing.slug);
        }
        onChange?.();
      },
      {
        onError: async () => setFavourite(!next),
      },
    );
  }

  const label = favourite
    ? t.marketplace.detail.unfavourite
    : t.marketplace.detail.favourite;

  return (
    <IconButton
      type="button"
      size="sm"
      variant="subtle"
      bg="bg.default"
      aria-label={label}
      title={label}
      aria-pressed={favourite}
      onClick={handleToggle}
    >
      {favourite ? <LikeSavedIcon /> : <LikeIcon />}
    </IconButton>
  );
}
