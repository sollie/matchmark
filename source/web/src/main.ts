import { mount } from "svelte";
import App from "./App.svelte";

const start = () => mount(App, { target: document.getElementById("app")! });

if (document.readyState === "complete") {
  start();
} else {
  window.addEventListener("load", start, { once: true });
}
