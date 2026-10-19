"use client";

import { getLogoURLs } from "@/lib/settings/logo";
import { Settings } from "@/lib/settings/settings";
import { useSettings } from "@/lib/settings/settings-client";
import { css, cx } from "@/styled-system/css";

type Props = {
  initialSettings?: Settings;
  alt: string;
  className?: string;
};

export function SiteLogo({ initialSettings, alt, className }: Props) {
  const result = useSettings(initialSettings);
  const logo = result.ready
    ? result.settings.metadata.logo
    : initialSettings?.metadata.logo;

  const { light, dark } = getLogoURLs(logo);

  if (light === dark) {
    return <img src={light} alt={alt} className={className} />;
  }

  return (
    <>
      <img
        src={light}
        alt={alt}
        className={cx(className, css({ _dark: { display: "none" } }))}
      />
      <img
        src={dark}
        alt={alt}
        className={cx(
          className,
          css({ display: "none", _dark: { display: "block" } }),
        )}
      />
    </>
  );
}
