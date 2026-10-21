import { AppRouter } from "./router";
import { Provider } from "@/components/ui/provider";
import { Toaster } from "@/components/ui/toaster";

/**
 * The root component of the application.
 * It sets up global providers and renders the main router.
 */
export default function App() {
  return (
    // Light only for now: the screens are designed and tested in light mode.
    <Provider defaultTheme="light" enableSystem={false}>
      <AppRouter />
      <Toaster />
    </Provider>
  );
}
