import React, { useState, useEffect } from "react";
import { Box, Flex, Text, Button, IconButton } from "@chakra-ui/react";
import { Input } from '@chakra-ui/react'
import { Alert } from '@chakra-ui/react'
import { Checkbox } from '@chakra-ui/react'
import { LuSearch } from "react-icons/lu"
import { useAuthStore } from "../store/auth";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../utils/api";

export default function LoginNew() {
  const setToken = useAuthStore((s) => s.setToken);
  const token = useAuthStore((s) => s.token);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (token) {
      navigate("/collections", { replace: true });
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
        throw new Error(json?.error?.message || "Login failed");
      }
      const token = json?.data?.token;
      if (!token) throw new Error("No token received from server");
      setToken(token);
      navigate("/collections");
    } catch (err: any) {
      setError(err.message || "Login failed");
    }
  };

  return (
    <Flex minH="100vh" bg="gray.50" align="center" justify="center" direction="column" position="relative">
      {/* Language selector top right */}
      <Box position="absolute" top="6" right="10">
        <Text fontSize="sm" color="gray.600" fontWeight="bold" cursor="pointer">
          English ▼
        </Text>
      </Box>

      <Box
        as="form"
        onSubmit={handleLogin}
        bg="white"
        p={{ base: 6, md: 10 }}
        rounded="xl"
        boxShadow="lg"
        minW={{ base: "90vw", sm: "400px", md: "430px" }}
        maxW="lg"
        mx="auto"
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

        {/* Email Field */}
        <Text fontWeight="semibold" color="gray.700" fontSize="sm" mb={1}>
          Email
        </Text>
        <Input
          mb={4}
          type="email"
          placeholder="you@email.com"
          value={username}
          onChange={e => setUsername(e.target.value)}
          size="lg"
          bg="gray.50"
          borderColor="gray.200"
          fontSize="md"
          required
        />

        {/* Password Field */}
        <Flex justify="space-between" align="center" mb={1}>
          <Text fontWeight="semibold" color="gray.700" fontSize="sm">
            Password
          </Text>
          <Text color="blue.500" fontSize="sm" fontWeight="medium" cursor="pointer">
            Forgot password?
          </Text>
        </Flex>
        <Box position="relative" mb={1}>
          <Input
            type={showPassword ? "text" : "password"}
            placeholder="**********"
            value={password}
            onChange={e => setPassword(e.target.value)}
            size="lg"
            bg="gray.50"
            borderColor="gray.200"
            fontSize="md"
            required
            pr={10}
          />
          <Box position="absolute" right={2} top="50%" transform="translateY(-50%)">
            <IconButton
              size="sm"
              variant="ghost"
              aria-label="Search"
              onClick={() => {}}
              tabIndex={0}
            >
              <LuSearch />
            </IconButton>
          </Box>
        </Box>

        {/* Remember me */}
        <Flex align="center" mb={5} mt={2}>
          <Checkbox.Root id="remember-me" mr={2}>
            <Checkbox.Indicator />
            <span style={{ marginLeft: 8 }}>Remember me</span>
          </Checkbox.Root>
        </Flex>

        <Button
          colorScheme="blue"
          size="lg"
          width="100%"
          fontWeight="bold"
          type="submit"
          mb={4}
        >
          Login
        </Button>

        {error && (
          <Alert.Root status="error" mt={2} rounded="md" fontSize="sm">
            <Alert.Indicator />
            <Alert.Content>
              <Alert.Title>Error</Alert.Title>
              <Alert.Description>{error}</Alert.Description>
            </Alert.Content>
          </Alert.Root>
        )}

        {/* Divider (manual, because Divider doesn't exist) */}
        <Flex align="center" my={6}>
          <Box flex="1" h="1px" bg="gray.100" />
          <Text mx={2} color="gray.400" fontSize="xs">OR LOGIN WITH</Text>
          <Box flex="1" h="1px" bg="gray.100" />
        </Flex>

        {/* SSO Buttons */}
        <Flex gap={4} justify="center" mt={2}>
          <Button
            variant="outline"
            w="40px"
            h="40px"
            p={0}
            borderRadius="md"
            border="1px solid #e5e7eb"
            aria-label="Login with Auth0"
          >
            <img src="/auth0-logo.svg" alt="Auth0" style={{ width: 22, height: 22 }} />
          </Button>
          <Button
            variant="outline"
            w="40px"
            h="40px"
            p={0}
            borderRadius="md"
            border="1px solid #e5e7eb"
            aria-label="Login with Okta"
          >
            <img src="/okta-logo.svg" alt="Okta" style={{ width: 22, height: 22 }} />
          </Button>
          <Button
            variant="outline"
            w="40px"
            h="40px"
            p={0}
            borderRadius="md"
            border="1px solid #e5e7eb"
            aria-label="More options"
          >
            <Text fontSize="2xl" color="gray.400" mx="auto">…</Text>
          </Button>
        </Flex>
      </Box>
    </Flex>
  );
}
