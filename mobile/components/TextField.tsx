import { StyleSheet, Text, TextInput, View, type TextInputProps } from 'react-native'

import { colors, radius, spacing, theme, typography } from '../theme'

interface TextFieldProps extends TextInputProps {
  label: string
  error?: string
}

/** A labelled text input, the mobile counterpart of the web app's Field. */
export function TextField({ label, error, style, ...input }: TextFieldProps) {
  return (
    <View style={styles.container}>
      <Text style={styles.label}>{label}</Text>
      <TextInput
        {...input}
        placeholderTextColor={colors.neutral400}
        style={[styles.input, error ? styles.inputError : null, style]}
      />
      {error ? (
        <Text style={styles.error} accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
    </View>
  )
}

const styles = StyleSheet.create({
  container: { marginBottom: spacing.lg },
  label: { ...typography.label, color: theme.textPrimary, marginBottom: spacing.xs },
  input: {
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    fontSize: 16,
    color: theme.textPrimary,
    backgroundColor: theme.surface,
  },
  inputError: { borderColor: theme.danger },
  error: { ...typography.caption, color: theme.dangerText, marginTop: spacing.xs },
})
