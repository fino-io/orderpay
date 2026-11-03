import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ command, mode }) => {
  const frontendDir = fileURLToPath(new URL(".", import.meta.url));
  const frontendEnv = loadEnv(mode, frontendDir, "VITE_");
  let apiBase = frontendEnv.VITE_API_BASE?.trim() || "";

  if (command === "serve" && !apiBase) {
    const backendDir = fileURLToPath(new URL("../backend/", import.meta.url));
    const port = loadEnv(mode, backendDir, "SERVER_HTTP_PORT").SERVER_HTTP_PORT;
    if (!port || !/^\d+$/.test(port) || Number(port) < 1 || Number(port) > 65535) {
      throw new Error("Set SERVER_HTTP_PORT (1-65535) in backend/.env before starting frontend development");
    }
    apiBase = `http://127.0.0.1:${port}`;
  }

  return {
    define: {
      "import.meta.env.VITE_API_BASE": JSON.stringify(apiBase),
    },
    plugins: [react()],
    server: {
      proxy: {
        "/monitor": {
          target: "http://127.0.0.1:3000",
          changeOrigin: true,
        },
      },
    },
    resolve: {
      alias: {
        "@": fileURLToPath(new URL(".", import.meta.url)),
      },
    },
    build: {
      outDir: fileURLToPath(new URL("../.fino/frontend/", import.meta.url)),
      emptyOutDir: true,
    },
  };
});
