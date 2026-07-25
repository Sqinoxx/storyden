"use client";

import { SliderValueChangeDetails } from "@ark-ui/react";

import { useTheme } from "@/lib/theme/ThemeProvider";

import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { RadioGroupField } from "@/components/ui/form/RadioGroupField";
import { Heading } from "@/components/ui/heading";
import { Slider } from "@/components/ui/slider";
import { css } from "@/styled-system/css";
import { Box, CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import {
  Props,
  useMemberInterfaceSettings,
} from "./useMemberInterfaceSettings";

function warmthLabel(value: number): string {
  if (value === 0) return "Neutral";
  if (value <= 20) return "Leicht warm";
  if (value <= 50) return "Warm";
  if (value <= 75) return "Sehr warm";
  return "Maximal warm";
}

export function MemberInterfaceSettings(props: Props) {
  const result = useMemberInterfaceSettings(props);
  const { resolvedTheme, warmth, setWarmth } = useTheme();

  if (!result.ready) {
    return <Unready />;
  }

  const { control, formState, onSubmit } = result;

  function handleWarmthChange(details: SliderValueChangeDetails) {
    setWarmth(details.value[0] ?? 0);
  }

  return (
    <styled.div width="full" display="flex" flexDirection="column" gap="4">
      {/* Main settings form */}
      <styled.form
        width="full"
        display="flex"
        flexDirection="column"
        gap="4"
        onSubmit={onSubmit}
      >
        <CardBox className={lstack()}>
          <WStack>
            <Heading size="md">Interface settings</Heading>
            <Button type="submit" loading={formState.isSubmitting}>
              Save
            </Button>
          </WStack>

          <FormControl>
            <FormLabel>Text editor style</FormLabel>
            <RadioGroupField
              control={control}
              name="editorMode"
              items={[
                { label: "Rich text", value: "richtext" },
                { label: "Markdown", value: "markdown" },
              ]}
            />
            <FormHelperText>
              Choose your preferred editor style for composing threads, replies
              and pages.
            </FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel>Sidebar default state</FormLabel>
            <RadioGroupField
              control={control}
              name="sidebarDefaultState"
              items={[
                { label: "Open", value: "open" },
                { label: "Closed", value: "closed" },
              ]}
            />
            <FormHelperText>
              Choose your preferred default state for the sidebar when you visit
              the site.
            </FormHelperText>
          </FormControl>
        </CardBox>
      </styled.form>

      {/* Colour warmth – only shown in Light Mode, live/instant effect */}
      {resolvedTheme === "light" && (
        <CardBox className={lstack()}>
          <Heading size="md">Light Mode: Farbwärme</Heading>

          <FormControl>
            <WStack alignItems="center">
              <FormLabel mb="0">Farbwärme</FormLabel>
              {/* Badge showing current warmth label */}
              <span
                className={css({
                  px: "3",
                  py: "1",
                  borderRadius: "full",
                  bg: "bg.subtle",
                  fontSize: "sm",
                  fontWeight: "medium",
                  color: "fg.subtle",
                  minW: "28",
                  textAlign: "center",
                })}
              >
                {warmthLabel(warmth)}
              </span>
            </WStack>

            {/* Gradient preview bar */}
            <Box
              w="full"
              h="3"
              borderRadius="full"
              mb="2"
              style={{
                background:
                  "linear-gradient(to right, #f8f8f8 0%, #fdf5e6 40%, #fde8b0 70%, #f5c842 100%)",
                border: "1px solid var(--colors-border-subtle)",
              }}
            />

            <Slider
              colorPalette="accent"
              min={0}
              max={100}
              step={1}
              value={[warmth]}
              onValueChange={handleWarmthChange}
              defaultValue={[0]}
            />

            <FormHelperText>
              Steuert die Farbwärme der Oberfläche im Light Mode. Nur im Light
              Mode aktiv. Die Einstellung wird sofort angewendet und
              automatisch gespeichert.
            </FormHelperText>
          </FormControl>

          {/* Quick preset swatches */}
          <HStack gap="3" flexWrap="wrap" pt="1">
            {[0, 25, 50, 75, 100].map((val) => (
              <Box
                key={val}
                onClick={() => setWarmth(val)}
                title={warmthLabel(val)}
                w="10"
                h="10"
                borderRadius="md"
                cursor="pointer"
                style={{
                  background: `hsl(${40 + val * 0.1}, ${10 + val * 0.4}%, ${98 - val * 0.06}%)`,
                  border:
                    warmth === val
                      ? "2px solid var(--colors-accent-default)"
                      : "2px solid var(--colors-border-subtle)",
                  transition: "border-color 0.15s, transform 0.15s",
                }}
              />
            ))}
            <Box fontSize="xs" color="fg.muted" alignSelf="center">
              Schnell-Presets
            </Box>
          </HStack>
        </CardBox>
      )}
    </styled.div>
  );
}
