import { ComponentProps, forwardRef } from "react";

import { styled } from "@/styled-system/jsx";

const Select = styled("select", {
  base: {
    width: "full",
    height: "9",
    px: "2",
    borderWidth: "thin",
    borderStyle: "solid",
    borderColor: "border.default",
    borderRadius: "l2",
    bg: "bg.default",
    color: "fg.default",
    fontSize: "sm",
  },
});

export type NativeSelectProps = ComponentProps<typeof Select> & {
  options: { value: string; label: string }[];
};

export const NativeSelect = forwardRef<HTMLSelectElement, NativeSelectProps>(
  function NativeSelect({ options, ...props }, ref) {
    return (
      <Select ref={ref} {...props}>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </Select>
    );
  },
);
