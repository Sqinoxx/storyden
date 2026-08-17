import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { Heading } from "@/components/ui/heading";
import { CardBox, Flex, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useAssetSettings } from "./useAssetSettings";

export function AssetSettingsForm(props: Props) {
  const { control, formState, onSubmit } = useAssetSettings(props);

  return (
    <styled.form
      width="full"
      display="flex"
      flexDirection="column"
      gap="4"
      onSubmit={onSubmit}
    >
      <CardBox className={lstack()}>
        <WStack>
          <Heading size="md">Upload settings</Heading>
          <Button type="submit" loading={formState.isSubmitting}>
            Save
          </Button>
        </WStack>

        <Flex
          flexDir={{
            base: "column",
            md: "row",
          }}
          gap="2"
        >
          <FormControl>
            <FormLabel>Maximum upload size (MB)</FormLabel>
            <NumberInputField
              control={control}
              name="maxUploadSizeMb"
              scrubber={true}
              min={1}
              max={1024}
              step={5}
            />
            <FormHelperText>
              The largest file, in megabytes, a member may upload in a single
              request — attachments, cover images, plugin archives and
              similar. Files larger than this are rejected immediately with
              an error before anything is stored. Raising this increases
              memory and bandwidth usage per upload, so keep it within what
              your server can comfortably handle.
            </FormHelperText>
          </FormControl>
        </Flex>
      </CardBox>
    </styled.form>
  );
}
