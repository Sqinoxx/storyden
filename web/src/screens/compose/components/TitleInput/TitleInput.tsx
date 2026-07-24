"use client";

import { Controller } from "react-hook-form";

import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { HeadingInput } from "@/components/ui/heading-input";
import { useTranslation } from "@/lib/i18n";

import { useTitleInput } from "./useTitleInput";

export function TitleInput() {
  const { control, fieldError } = useTitleInput();
  const t = useTranslation();

  return (
    <>
      <FormControl>
        <Controller
          render={({ field: { onChange, ...field }, formState }) => {
            return (
              <HeadingInput
                id="title-input"
                placeholder={t.editor.titlePlaceholder}
                onValueChange={onChange}
                defaultValue={formState.defaultValues?.["title"]}
                {...field}
              />
            );
          }}
          control={control}
          name="title"
        />

        <FormErrorText>{fieldError?.message?.toString()}</FormErrorText>
      </FormControl>
    </>
  );
}
