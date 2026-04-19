import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";

import { App } from "./app/App";
import { PreferencesProvider } from "./lib/preferences";
import { queryClient } from "./lib/queryClient";
import { currentRouterBasename } from "./lib/runtime";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <PreferencesProvider>
        <BrowserRouter basename={currentRouterBasename()}>
          <App />
        </BrowserRouter>
      </PreferencesProvider>
    </QueryClientProvider>
  </React.StrictMode>,
);
