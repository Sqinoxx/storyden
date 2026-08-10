import type { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: [
        "/admin",
        "/settings",
        "/drafts",
        "/notifications",
        "/queue",
        "/reports",
        "/roles",
        "/robots/",
        "/new",
        "/search",
        "/login",
        "/logout",
        "/register",
        "/password-reset",
        "/oauth",
        "/auth",
        "/invitation",
        "/xdev",
        "/client-ip-test",
        "/api/",
      ],
    },
  };
}
