import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { Heading } from "@/components/ui/heading";
import { CardBox, Flex, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useAdminSessionSettings } from "./useSessionSettings";

export function SessionSettingsForm(props: Props) {
  const { control, formState, onSubmit } = useAdminSessionSettings(props);

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
          <Heading size="md">Session settings</Heading>
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
            <FormLabel>Inactivity timeout (minutes)</FormLabel>
            <NumberInputField
              control={control}
              name="idleTimeoutMinutes"
              scrubber={true}
              min={1}
              max={43_200}
              step={15}
            />
            <FormHelperText>
              How long a member stays signed in without &quot;remember me&quot;.
              The window slides forward while they are active, so this is a
              timeout on inactivity rather than a hard limit from sign-in.
              Closing the browser also ends these sessions.
            </FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel>Remember me default (days)</FormLabel>
            <NumberInputField
              control={control}
              name="rememberMeDefaultDays"
              scrubber={true}
              min={1}
              max={365}
              step={1}
            />
            <FormHelperText>
              Session length for members who tick &quot;remember me&quot; and
              have not chosen their own duration.
            </FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel>Remember me maximum (days)</FormLabel>
            <NumberInputField
              control={control}
              name="rememberMeMaxDays"
              scrubber={true}
              min={1}
              max={365}
              step={1}
            />
            <FormHelperText>
              Upper bound for member-chosen durations. Longer requests are
              clamped to this.
            </FormHelperText>
          </FormControl>
        </Flex>
      </CardBox>
    </styled.form>
  );
}
