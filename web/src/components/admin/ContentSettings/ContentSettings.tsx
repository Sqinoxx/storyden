import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { Heading } from "@/components/ui/heading";
import { CardBox, Flex, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useContentSettings } from "./useContentSettings";

export function ContentSettingsForm(props: Props) {
  const { control, formState, onSubmit } = useContentSettings(props);

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
          <Heading size="md">Content settings</Heading>
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
            <FormLabel>Replies per page</FormLabel>
            <NumberInputField
              control={control}
              name="repliesPerPage"
              scrubber={true}
              min={1}
              max={200}
              step={5}
            />
            <FormHelperText>
              How many replies are shown on each page of a thread. Lower values
              keep long threads fast to load. Changing this alters which page a
              reply lives on, so existing links to a specific page of a thread
              will point elsewhere.
            </FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel>Threads per page</FormLabel>
            <NumberInputField
              control={control}
              name="threadsPerPage"
              scrubber={true}
              min={1}
              max={100}
              step={5}
            />
            <FormHelperText>
              How many threads are shown on each page of the feed and category
              listings.
            </FormHelperText>
          </FormControl>
        </Flex>
      </CardBox>
    </styled.form>
  );
}
