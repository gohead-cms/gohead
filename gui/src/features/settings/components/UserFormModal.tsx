import { type FormEvent, useState } from "react";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  VStack,
} from "@chakra-ui/react";
import { api } from "../../../services/api";
import { errorMessage } from "../../../shared/utils/errors";
import { RoleSelect } from "./RoleSelect";
import { MIN_PASSWORD_LENGTH, isEmail, userRoleName } from "./settingsFormat";
import { type AdminUser, ROLES, type RoleName } from "./types";

function asRole(name: string): RoleName {
  return (ROLES as readonly string[]).includes(name) ? (name as RoleName) : "viewer";
}

/**
 * Creates a user (user = null) or edits one's email, role and password.
 * Mount it with a key per user so its fields start from that user.
 */
export function UserFormModal({
  isOpen,
  user,
  isSelf,
  onClose,
  onSaved,
}: {
  isOpen: boolean;
  user: AdminUser | null;
  isSelf: boolean;
  onClose: () => void;
  onSaved: (message: string) => void;
}) {
  const creating = user === null;
  const initialRole = user ? asRole(userRoleName(user)) : "editor";
  const [username, setUsername] = useState(user?.username ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<RoleName>(initialRole);
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const usernameError = !creating
    ? null
    : !username.trim()
      ? "Username is required"
      : username.includes(":")
        ? "Username cannot contain ':'"
        : null;
  const emailError = !email.trim() ? "Email is required" : !isEmail(email) ? "This does not look like an email address" : null;
  const passwordError =
    creating && !password
      ? "Password is required"
      : password && password.length < MIN_PASSWORD_LENGTH
        ? `At least ${MIN_PASSWORD_LENGTH} characters`
        : null;
  const hasErrors = !!(usernameError || emailError || passwordError);

  const changes: Record<string, string> = {};
  if (user) {
    if (email.trim() !== user.email) changes.email = email.trim();
    if (role !== initialRole && !isSelf) changes.role_name = role;
    if (password) changes.password = password;
  }
  const unchanged = !creating && Object.keys(changes).length === 0;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (hasErrors || unchanged) return;
    setSaving(true);
    setServerError(null);
    try {
      if (user) {
        await api("PUT", `/admin/users/${user.ID}`, changes);
        onSaved(`User ${user.username} updated`);
      } else {
        await api("POST", "/admin/users", {
          username: username.trim(),
          email: email.trim(),
          password,
          role_name: role,
        });
        onSaved(`User ${username.trim()} created`);
      }
    } catch (err) {
      setServerError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} closeOnOverlayClick={!saving}>
      <ModalOverlay />
      <ModalContent>
        <form onSubmit={submit} noValidate>
          <ModalHeader>{creating ? "New user" : `Edit ${user.username}`}</ModalHeader>
          <ModalCloseButton isDisabled={saving} />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              {serverError && (
                <Alert status="error" borderRadius="md">
                  <AlertIcon />
                  <AlertDescription>{serverError}</AlertDescription>
                </Alert>
              )}
              {creating && (
                <FormControl isRequired isInvalid={submitted && !!usernameError}>
                  <FormLabel>Username</FormLabel>
                  <Input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="off" autoFocus />
                  <FormErrorMessage>{usernameError}</FormErrorMessage>
                </FormControl>
              )}
              <FormControl isRequired isInvalid={submitted && !!emailError}>
                <FormLabel>Email</FormLabel>
                <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" />
                <FormErrorMessage>{emailError}</FormErrorMessage>
              </FormControl>
              <FormControl isRequired={creating} isInvalid={submitted && !!passwordError}>
                <FormLabel>{creating ? "Password" : "New password"}</FormLabel>
                <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
                {submitted && passwordError ? (
                  <FormErrorMessage>{passwordError}</FormErrorMessage>
                ) : (
                  <FormHelperText>
                    {creating
                      ? `At least ${MIN_PASSWORD_LENGTH} characters. Share it with the user through a safe channel.`
                      : "Leave empty to keep the current password."}
                  </FormHelperText>
                )}
              </FormControl>
              <RoleSelect
                value={role}
                onChange={setRole}
                isDisabled={isSelf}
                disabledReason="You cannot change your own role: another admin must do it, so the site always keeps an admin."
              />
            </VStack>
          </ModalBody>
          <ModalFooter gap={3}>
            <Button onClick={onClose} isDisabled={saving}>
              Cancel
            </Button>
            <Button type="submit" colorScheme="blue" isLoading={saving} isDisabled={unchanged}>
              {creating ? "Create user" : "Save changes"}
            </Button>
          </ModalFooter>
        </form>
      </ModalContent>
    </Modal>
  );
}
