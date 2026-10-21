import { type FormEvent, useState } from "react";
import { Alert, Button, Input, VStack } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { api } from "../../../services/api";
import { errorMessage } from "../../../shared/utils/errors";
import { RoleSelect } from "./RoleSelect";
import { MIN_PASSWORD_LENGTH, isEmail, userRoleName } from "./settingsFormat";
import type { AdminUser } from "./types";

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
  const initialRole = user ? userRoleName(user) || "viewer" : "editor";
  const [username, setUsername] = useState(user?.username ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState(initialRole);
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
    <DialogRoot
      open={isOpen}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      closeOnInteractOutside={!saving}
    >
      <DialogContent>
        <form onSubmit={submit} noValidate>
          <DialogHeader>
            <DialogTitle>{creating ? "New user" : `Edit ${user.username}`}</DialogTitle>
          </DialogHeader>
          <DialogCloseTrigger disabled={saving} />
          <DialogBody>
            <VStack gap={4} align="stretch">
              {serverError && (
                <Alert.Root role="alert" status="error" borderRadius="md">
                  <Alert.Indicator />
                  <Alert.Content>
                    <Alert.Description>{serverError}</Alert.Description>
                  </Alert.Content>
                </Alert.Root>
              )}
              {creating && (
                <Field label="Username" required invalid={submitted && !!usernameError} errorText={usernameError}>
                  <Input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="off" autoFocus />
                </Field>
              )}
              <Field label="Email" required invalid={submitted && !!emailError} errorText={emailError}>
                <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" />
              </Field>
              <Field
                label={creating ? "Password" : "New password"}
                required={creating}
                invalid={submitted && !!passwordError}
                errorText={passwordError}
                helperText={
                  submitted && passwordError
                    ? undefined
                    : creating
                      ? `At least ${MIN_PASSWORD_LENGTH} characters. Share it with the user through a safe channel.`
                      : "Leave empty to keep the current password."
                }
              >
                <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
              </Field>
              <RoleSelect
                value={role}
                onChange={setRole}
                isDisabled={isSelf}
                disabledReason="You cannot change your own role: another admin must do it, so the site always keeps an admin."
              />
            </VStack>
          </DialogBody>
          <DialogFooter gap={3}>
            <Button variant="subtle" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button type="submit" colorPalette="blue" loading={saving} disabled={unchanged}>
              {creating ? "Create user" : "Save changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </DialogRoot>
  );
}
