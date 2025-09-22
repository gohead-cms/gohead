import React, { useState, useEffect } from "react";
import {
  Box,
  Button,
  Input,
  Heading,
  Alert,
  Flex,
  Text,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  VStack,
  FormControl,
  FormLabel,
} from "@chakra-ui/react";
import { useAuthStore } from "../../services/auth";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../../services/api";

export default function Login() {
  const setToken = useAuthStore((s) => s.setToken);
  const token = useAuthStore((s) => s.token);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (token) {
      navigate("/workspace", { replace: true });
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
      navigate("/workspace");
    } catch (err: any) {
      setError(err.message || "Login failed");
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
          <VStack spacing={4}>
            <FormControl>
              <FormLabel>Username</FormLabel>
              <Input
                placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoFocus
              />
            </FormControl>
            <FormControl>
              <FormLabel>Password</FormLabel>
              <Input
                placeholder="Password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </FormControl>
            <Button type="submit" colorScheme="blue" width="100%">
              Login
            </Button>
          </VStack>
        </form>
        {error && (
          <Alert status="error" mt={4} borderRadius="md">
            <AlertIcon />
            <Box>
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Box>
          </Alert>
        )}
      </Box>
    </Flex>
  );
}
