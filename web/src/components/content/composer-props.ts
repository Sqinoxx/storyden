import { Asset } from "@/api/openapi-schema";

export type ContentComposerProps = {
  className?: string;
  disabled?: boolean;
  resetKey?: string;
  initialValue?: string;
  initialValueFormat?: "html" | "markdown";

  // NOTE: This is not for making the editor controllable but for optimistic
  // mutation/revalidation of disabled editors. Use with care!
  value?: string;
  placeholder?: string;
  onChange?: (value: string, isEmpty: boolean) => void;
  onAssetUpload?: (asset: Asset) => void;

  // Submitting while an upload is in flight posts the body without the file,
  // so consumers use this to hold their submit action until it settles.
  onUploadingChange?: (uploading: boolean) => void;

  // Consumers that render their own attachment list (the full compose screen)
  // keep non-image uploads out of the document and show them beside it. Ones
  // without such a list need them embedded, or a dropped document uploads and
  // then has nothing to show for it.
  inlineAttachments?: boolean;

  // Stretches the editor to fill a tall parent container (e.g. the compose
  // screen's card) instead of only hugging its typed content height.
  fillHeight?: boolean;
};
