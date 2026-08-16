"use client";

import type { UseFormRegisterReturn } from "react-hook-form";

import { SEMESTER_OPTIONS, formatSemester } from "@/lib/profile/academic";
import { styled } from "@/styled-system/jsx";

type Props = {
  register: UseFormRegisterReturn;
};

export function SemesterField({ register }: Props) {
  return (
    <styled.select
      aria-label="Fachsemester"
      width="full"
      height="9"
      px="2"
      borderWidth="thin"
      borderStyle="solid"
      borderColor="border.default"
      borderRadius="l2"
      bg="bg.default"
      color="fg.default"
      textAlign="center"
      required
      defaultValue=""
      {...register}
    >
      <option value="" disabled>
        Fachsemester
      </option>
      {SEMESTER_OPTIONS.map((value) => (
        <option key={value} value={value}>
          {formatSemester(value)}
        </option>
      ))}
    </styled.select>
  );
}
