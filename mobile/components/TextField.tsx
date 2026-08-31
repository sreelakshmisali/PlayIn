import { useState } from 'react'
import { StyleSheet, Text, TextInput, View, type TextInputProps } from 'react-native'

import { colors, inputPresets, spacing, theme, typography } from '../theme'

interface TextFieldProps extends TextInputProps {
  label: string
  error?: string
}

/** A labelled text input, the mobile counterpart of the web app's Field.
 * Pass `editable={false}` for a read-only/disabled field — it gets its own
 * muted visual treatment rather than looking like an active input. */
export function TextField({ label, error, style, editable, ...input }: TextFieldProps) {
  const disabled = editable === false
  const [focused, setFocused] = useState(false)

  return (
    <View style={styles.container}>
      <Text style={[styles.label, disabled && styles.labelDisabled]}>{label}</Text>
      <TextInput
        {...input}
        editable={editable}
        onFocus={(e) => {
          setFocused(true)
          input.onFocus?.(e)
        }}
        onBlur={(e) => {
          setFocused(false)
          input.onBlur?.(e)
        }}
        placeholderTextColor={colors.neutral400}
        style={[
          styles.input,
          disabled && inputPresets.disabled,
          focused && !disabled && !error && inputPresets.focused,
          error ? inputPresets.error : null,
          style,
        ]}
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
  container: { marginBottom: spacing.xl },
  label: { ...typography.label, color: theme.textPrimary, marginBottom: spacing.xs },
  labelDisabled: { color: theme.textDisabled },
  input: inputPresets.field,
  error: { ...typography.caption, color: theme.dangerText, marginTop: spacing.xs },
})
