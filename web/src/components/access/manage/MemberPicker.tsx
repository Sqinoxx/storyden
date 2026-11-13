"use client";

import { debounce } from "lodash";
import { useEffect, useMemo, useState } from "react";

import { profileList } from "@/api/openapi-client/profiles";
import { ProfileReference } from "@/api/openapi-schema";
import { MemberAvatar } from "@/components/member/MemberBadge/MemberAvatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  onChange: (handle: string) => void;
};

// Finds a member by name or username. Typing a username and not picking a
// suggestion still works, for people the search does not surface.
export function MemberPicker({ onChange }: Props) {
  const t = useTranslation();
  const m = t.access.manage;

  const [query, setQuery] = useState("");
  const [results, setResults] = useState<ProfileReference[] | undefined>();
  const [selected, setSelected] = useState<ProfileReference | undefined>();

  const search = useMemo(
    () =>
      debounce(async (q: string) => {
        try {
          const res = await profileList({ q });
          setResults(res.profiles.slice(0, 6));
        } catch {
          setResults([]);
        }
      }, 250),
    [],
  );

  useEffect(() => () => search.cancel(), [search]);

  function onType(value: string) {
    setQuery(value);
    onChange(value.trim().replace(/^@/, ""));

    const q = value.trim().replace(/^@/, "");
    if (q.length < 2) {
      search.cancel();
      setResults(undefined);
      return;
    }
    search(q);
  }

  function pick(p: ProfileReference) {
    search.cancel();
    setSelected(p);
    setResults(undefined);
    onChange(p.handle);
  }

  function clear() {
    setSelected(undefined);
    setQuery("");
    onChange("");
  }

  if (selected) {
    return (
      <HStack
        gap="2"
        justify="space-between"
        borderWidth="thin"
        borderColor="border.default"
        borderRadius="sm"
        p="2"
        data-testid="access-member-selected"
      >
        <HStack gap="2" minW="0">
          <MemberAvatar profile={selected} size="sm" />
          <styled.span fontWeight="semibold" overflowWrap="anywhere">
            {selected.name}
          </styled.span>
          <styled.span color="fg.muted" fontSize="sm">
            @{selected.handle}
          </styled.span>
        </HStack>
        <Button type="button" size="xs" variant="ghost" onClick={clear}>
          {m.memberChange}
        </Button>
      </HStack>
    );
  }

  return (
    <LStack gap="1" w="full">
      <Input
        id="access-grant-handle"
        aria-label={m.memberSearch}
        placeholder={m.memberSearchPlaceholder}
        fontSize={{ base: "md", md: "sm" }}
        value={query}
        onChange={(e) => onType(e.currentTarget.value)}
        autoComplete="off"
      />
      {results && (
        <LStack
          gap="0"
          w="full"
          borderWidth="thin"
          borderColor="border.default"
          borderRadius="sm"
          role="listbox"
          aria-label={m.memberSearch}
        >
          {results.length === 0 ? (
            <styled.p p="2" fontSize="sm" color="fg.muted">
              {m.memberSearchEmpty}
            </styled.p>
          ) : (
            results.map((p) => (
              <styled.button
                key={p.id}
                type="button"
                role="option"
                aria-selected={false}
                display="flex"
                alignItems="center"
                gap="2"
                w="full"
                p="2"
                textAlign="left"
                cursor="pointer"
                _hover={{ bg: "bg.muted" }}
                onClick={() => pick(p)}
              >
                <MemberAvatar profile={p} size="sm" />
                <styled.span fontWeight="medium">{p.name}</styled.span>
                <styled.span color="fg.muted" fontSize="sm">
                  @{p.handle}
                </styled.span>
              </styled.button>
            ))
          )}
        </LStack>
      )}
    </LStack>
  );
}
