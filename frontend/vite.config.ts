import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// 开发环境通过代理把 /agui 转发到后端，避免跨域问题。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/agui": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
});
