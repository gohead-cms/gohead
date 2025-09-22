import AppRouter from "./router";
import { ChakraProvider } from "@chakra-ui/react"; // Example provider

/**
 * The root component of the application.
 * It sets up global providers and renders the main router.
 */
export default function App() {
  return (
    <ChakraProvider>
      <AppRouter />
    </ChakraProvider>
  );
}
