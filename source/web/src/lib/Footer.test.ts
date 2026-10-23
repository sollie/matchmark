import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import Footer from "./Footer.svelte";
import { stubFetch } from "../test/fetch";

describe("Footer", () => {
  it("shows copyright, build version and repository link", async () => {
    stubFetch((c) => (c.path === "/api/version" ? { body: { version: "v1.2.3" } } : undefined));
    render(Footer);
    expect(await screen.findByText("MatchMark v1.2.3")).toBeTruthy();
    expect(screen.getByText("© 2026 Pål Sollie")).toBeTruthy();
    const link = screen.getByRole("link", { name: "Source code on GitHub" });
    expect(link.getAttribute("href")).toBe("https://github.com/sollie/matchmark");
    expect(link.getAttribute("rel")).toBe("noopener noreferrer");
    expect(link.getAttribute("target")).toBe("_blank");
  });

  it("omits the version when the request fails", async () => {
    stubFetch(() => ({ status: 500 }));
    render(Footer);
    expect(await screen.findByText("MatchMark")).toBeTruthy();
  });
});
