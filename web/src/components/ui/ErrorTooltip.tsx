import { Portal } from "@ark-ui/react";

import { WarningIcon } from "@/components/ui/icons/Warning";
import * as Popover from "@/components/ui/popover";
import { useTranslation } from "@/lib/i18n";
import { deriveError } from "@/utils/error";

import { IconButton } from "./icon-button";

type Props = {
  error?: unknown;
};

export function ErrorTooltip({ error }: Props) {
  const t = useTranslation();
  if (!error) {
    return null;
  }

  const errorMessage = deriveError(error);

  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <IconButton
          type="button"
          size="xs"
          variant="ghost"
          aria-label={t.common.showErrorDetails}
        >
          <WarningIcon color="fg.error" />
        </IconButton>
      </Popover.Trigger>
      <Portal>
        <Popover.Positioner>
          <Popover.Content>
            <Popover.Arrow>
              <Popover.ArrowTip />
            </Popover.Arrow>
            {errorMessage}
          </Popover.Content>
        </Popover.Positioner>
      </Portal>
    </Popover.Root>
  );
}
