"use client";

import { CommandDock } from "@/components/site/CommandDock/CommandDock";
import { ButtonProps } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { MenuIcon } from "@/components/ui/icons/Menu";
import { SiteIcon } from "@/components/ui/icons/Site";
import { WStack } from "@/styled-system/jsx";

import { CloseAction } from "../../Action/Close";
import { AccountMenu } from "../AccountMenu/AccountMenu";
import { ComposeAnchor } from "../Anchors/Compose";
import { HomeAnchor } from "../Anchors/Home";
import { LibraryAnchor } from "../Anchors/Library";
import { LoginAnchor } from "../Anchors/Login";
import { SearchAnchor } from "../Anchors/Search";
import { ContentNavigationList } from "../ContentNavigationList/ContentNavigationList";

import { useMobileCommandBar } from "./useMobileCommandBar";
import { useTranslation } from "@/lib/i18n";

type Props = {
  canRegister?: boolean;
};

export function MobileCommandBar({ canRegister }: Props) {
  const {
    isExpanded,
    onExpand,
    onClose,
    account,
    isAccountMenuOpen,
    onAccountMenuOpenChange,
  } = useMobileCommandBar();

  function handleAccountMenuOpenChange(details: { open: boolean }) {
    onAccountMenuOpenChange(details.open);
  }

  return (
    <CommandDock
      isOpen={isExpanded}
      onClickOutside={onClose}
      render={() => {
        return <ContentNavigationList />;
      }}
    >
      <WStack alignItems="center">
        {isExpanded ? (
          <>
            {account ? (
              <AccountMenu
                account={account}
                size="sm"
                open={isAccountMenuOpen}
                onOpenChange={handleAccountMenuOpenChange}
                closeOnThemeChange
              />
            ) : (
              <SiteIcon borderRadius="md" w="8" h="8" />
            )}
            {account && <LibraryAnchor hideLabel size="sm" />}
            <CloseAction onClick={onClose} size="sm" />
          </>
        ) : (
          <>
            {account ? (
              <AccountMenu
                account={account}
                size="sm"
                open={isAccountMenuOpen}
                onOpenChange={handleAccountMenuOpenChange}
                closeOnThemeChange
              />
            ) : (
              <SiteIcon borderRadius="md" w="8" h="8" />
            )}
            <HomeAnchor hideLabel size="sm" />
            {account ? (
              <ComposeAnchor hideLabel size="sm" />
            ) : (
              <LoginAnchor />
            )}
            <SearchAnchor hideLabel size="sm" />
            {account && <ExpandTrigger onClick={onExpand} />}
          </>
        )}
      </WStack>
    </CommandDock>
  );
}

function ExpandTrigger(props: ButtonProps) {
  const t = useTranslation();
  return (
    <IconButton
      title={t.nav.mainMenu}
      variant="ghost"
      size="sm"
      {...props}
    >
      <MenuIcon />
    </IconButton>
  );
}
