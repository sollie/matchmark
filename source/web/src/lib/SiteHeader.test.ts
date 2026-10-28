import { afterEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/svelte";
import SiteHeader from "./SiteHeader.svelte";
import { router } from "./router.svelte";

afterEach(() => {
  router.path = "/";
});

describe("SiteHeader", () => {
  it("links to the admin area from public routes", () => {
    render(SiteHeader);
    const link = screen.getByRole("link", { name: "Sign in" });
    expect(link.getAttribute("href")).toBe("/admin");
    expect(link.getAttribute("title")).toBe("Sign in");
  });

  it("shows the icon on results routes", () => {
    router.path = "/results/3";
    render(SiteHeader);
    expect(screen.getByRole("link", { name: "Sign in" })).toBeTruthy();
  });

  it("hides the icon on the scorer route", () => {
    router.path = "/score";
    render(SiteHeader);
    expect(screen.queryByRole("link", { name: "Sign in" })).toBeNull();
  });
});
