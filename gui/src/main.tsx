import { ChakraProvider } from "@chakra-ui/react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    {/* ChakraProvider will use its default theme automatically */}
    <ChakraProvider>
      <App />
    </ChakraProvider>
  </React.StrictMode>
);