import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";

import { useLinkInput } from "./useLinkInput";

export function LinkInput() {
  const t = useTranslation();
  const { register, fieldError } = useLinkInput();

  return (
    <FormControl>
      <Input
        size="xs"
        placeholder={t.editor.linkPlaceholder}
        type="url"
        {...register("url")}
      />
      <FormErrorText>{fieldError?.message}</FormErrorText>
    </FormControl>
  );
}
