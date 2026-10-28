import { FormEvent } from "react";

export function blockSubmitWhileUploading(
  isUploading: boolean,
  onSubmit: (e: FormEvent<HTMLFormElement>) => unknown,
) {
  return (e: FormEvent<HTMLFormElement>) => {
    if (isUploading) {
      e.preventDefault();
      return;
    }

    return onSubmit(e);
  };
}
