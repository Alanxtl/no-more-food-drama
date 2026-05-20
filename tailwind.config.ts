import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#25231f",
        paper: "#fbfaf7",
        line: "#e7e2d8",
        accent: "#2f7d6d",
        danger: "#b64b4b"
      }
    }
  },
  plugins: []
};

export default config;
