import React, { useState, useEffect } from "react";
import {
  Box,
  Button,
  Input,
  Alert,
  Flex,
  Text,
  VStack,
  Field,
} from "@chakra-ui/react";
import { useAuthStore } from "../../services/auth";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../../services/api";
import { resetLabel, useDemoInfo } from "../demo/demo";
import { startSSO, takeSSOResult, useSSOProviders } from "./sso";

export function LoginPage() {
  const setToken = useAuthStore((s) => s.setToken);
  const token = useAuthStore((s) => s.token);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  // Back from a provider: the token or the reason, read once.
  const [sso] = useState(takeSSOResult);
  const [error, setError] = useState<string | null>(sso?.error ?? null);
  const navigate = useNavigate();
  const demo = useDemoInfo();
  const providers = useSSOProviders();

  useEffect(() => {
    if (sso?.token) setToken(sso.token);
  }, [sso, setToken]);

  // Signed in: go on, unless a provider sign-in (or linking) failed, which
  // stays on screen.
  useEffect(() => {
    if (token && !sso?.error) {
      navigate(sso?.returnPath ?? "/dashboard", { replace: true });
    }
  }, [token, navigate, sso]);

  const signIn = async (credentials: { username: string; password: string }) => {
    setError(null);
    try {
      const res = await apiFetch("/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(credentials),
      });
      const json = await res.json();

      if (!res.ok) {
        // Backend puts error message at json.error.message
        throw new Error(json?.error?.message || "Login failed");
      }
      const token = json?.data?.token;
      if (!token) throw new Error("No token received from server");
      setToken(token);
      navigate("/dashboard");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Login failed");
    }
  };

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    void signIn({ username, password });
  };

  return (
    <Flex minH="100vh" align="center" justify="center" bg="gray.50">
      <Box
        maxW="sm"
        mx="auto"
        p={8}
        bg="white"
        boxShadow="lg"
        borderRadius="md"
      >
        <Flex direction="column" align="center" mb={8}>
          <img
            src="/gohead_logo.svg"
            alt="GoHead Logo"
            style={{ height: 56, width: 56, marginBottom: 24 }}
          />
          <Text fontSize="2xl" fontWeight="bold" mb={1}>
            Welcome back!
          </Text>
          <Text fontSize="md" color="gray.500" mb={2}>
            Log in to your GoHead account
          </Text>
        </Flex>
        {demo && (
          <Box borderWidth="1px" borderColor="purple.200" bg="purple.50" borderRadius="md" p={4} mb={6} data-testid="demo-login">
            <Text fontWeight="semibold" mb={1}>
              Try the demo
            </Text>
            <Text fontSize="sm" color="gray.700" mb={3}>
              Sign in as <code>{demo.username}</code> with the password <code>{demo.password}</code> to look around, write drafts
              and try agents.{demo.resetAt && ` Content is reset every day at ${resetLabel(demo)}.`}
            </Text>
            <Button size="sm" colorPalette="purple" width="100%" onClick={() => void signIn({ username: demo.username, password: demo.password })}>
              Sign in to the demo
            </Button>
          </Box>
        )}
        <form onSubmit={handleLogin}>
          <VStack gap={4}>
            <Field.Root>
              <Field.Label>Username</Field.Label>
              <Input
                placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoFocus
              />
            </Field.Root>
            <Field.Root>
              <Field.Label>Password</Field.Label>
              <Input
                placeholder="Password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </Field.Root>
            <Button type="submit" colorPalette="blue" width="100%">
              Login
            </Button>
          </VStack>
        </form>
        {providers.length > 0 && (
          <VStack gap={2} mt={4}>
            <Text fontSize="sm" color="gray.500">
              or
            </Text>
            {providers.map((p) => (
              <Button key={p.id} variant="outline" width="100%" onClick={() => startSSO(p.id)}>
                Sign in with {p.label}
              </Button>
            ))}
          </VStack>
        )}
        {error && (
          <Alert.Root role="alert" status="error" mt={4} borderRadius="md">
            <Alert.Indicator />
            <Alert.Content>
              <Alert.Title>Error</Alert.Title>
              <Alert.Description>{error}</Alert.Description>
            </Alert.Content>
          </Alert.Root>
        )}
        {token && sso?.error && (
          <Button mt={3} variant="subtle" width="100%" onClick={() => navigate("/dashboard", { replace: true })}>
            Back to GoHead
          </Button>
        )}
      </Box>
    </Flex>
  );
}
