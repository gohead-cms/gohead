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

export function LoginPage() {
  const setToken = useAuthStore((s) => s.setToken);
  const token = useAuthStore((s) => s.token);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (token) {
      navigate("/dashboard", { replace: true });
    }
  }, [token, navigate]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      const res = await apiFetch("/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
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
        {error && (
          <Alert.Root role="alert" status="error" mt={4} borderRadius="md">
            <Alert.Indicator />
            <Alert.Content>
              <Alert.Title>Error</Alert.Title>
              <Alert.Description>{error}</Alert.Description>
            </Alert.Content>
          </Alert.Root>
        )}
      </Box>
    </Flex>
  );
}
