// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightThemeFlexoki from "starlight-theme-flexoki";
import lucode from "lucode-starlight";
import mermaid from "astro-mermaid";

// https://astro.build/config
export default defineConfig({
  site: "https://sotnii.github.io",
  base: "/raijin",
  integrations: [
    mermaid({
      theme: "forest",
      autoTheme: true,
    }),
    starlight({
      title: "raijin",
      customCss: ["./src/styles/custom.css"],
      titleDelimiter: "|",
      favicon: "/favicon.svg",
      logo: {
        dark: "./src/assets/rjn-dark.svg",
        light: "./src/assets/rjn-light.svg",
      },
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/sotnii/raijin",
        },
      ],
      plugins: [
        lucode({
          docs: {
            includeAiUtilities: true,
          },
        }),
      ],
      sidebar: [
        {
          label: "Start here",
          items: ["about", "getting-started", "examples"],
        },
      ],
    }),
  ],
});
