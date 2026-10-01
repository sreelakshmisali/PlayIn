import { useState, type ReactNode } from 'react'
import {
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  type StyleProp,
  type TextInputProps,
  type ViewStyle,
} from 'react-native'
import { Ionicons } from '@expo/vector-icons'

import { colors, inputPresets, radius, spacing, theme, typography } from '../theme'

interface TextFieldProps extends TextInputProps {
  label: string
  error?: string
  containerStyle?: StyleProp<ViewStyle>
  leftIcon?: ReactNode
  rightIcon?: ReactNode
  /** Enable toggle password visibility icon when secureTextEntry is true. Defaults to true. */
  showVisibilityToggle?: boolean
}

/**
 * A labelled, refined text input field.
 *
 * Supports optional leading/trailing icons, focus border highlighting,
 * and built-in password visibility toggle with standard sports/lifestyle aesthetics.
 */
export function TextField({
  label,
  error,
  style,
  containerStyle,
  editable,
  secureTextEntry,
  leftIcon,
  rightIcon,
  showVisibilityToggle = true,
  ...input
}: TextFieldProps) {
  const disabled = editable === false
  const [focused, setFocused] = useState(false)
  const [hiddenPassword, setHiddenPassword] = useState(secureTextEntry ?? false)

  const isPasswordField = Boolean(secureTextEntry && showVisibilityToggle)
  const isSecure = isPasswordField ? hiddenPassword : secureTextEntry

  return (
    <View style={[styles.container, containerStyle]}>
      <Text style={[styles.label, disabled && styles.labelDisabled]}>{label}</Text>

      <View
        style={[
          styles.inputWrapper,
          disabled && inputPresets.disabled,
          focused && !disabled && !error && styles.wrapperFocused,
          error ? styles.wrapperError : null,
          style as ViewStyle,
        ]}
      >
        {leftIcon ? <View style={styles.leftIconWrapper}>{leftIcon}</View> : null}

        <TextInput
          {...input}
          editable={editable}
          secureTextEntry={isSecure}
          onFocus={(e) => {
            setFocused(true)
            input.onFocus?.(e)
          }}
          onBlur={(e) => {
            setFocused(false)
            input.onBlur?.(e)
          }}
          placeholderTextColor={colors.neutral400}
          style={[styles.textInput, disabled && styles.textInputDisabled]}
        />

        {isPasswordField ? (
          <Pressable
            onPress={() => setHiddenPassword((prev) => !prev)}
            accessibilityRole="button"
            accessibilityLabel={hiddenPassword ? 'Show password' : 'Hide password'}
            hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}
            style={styles.rightIconWrapper}
          >
            <Ionicons
              name={hiddenPassword ? 'eye-outline' : 'eye-off-outline'}
              size={18}
              color={theme.textSecondary}
            />
          </Pressable>
        ) : rightIcon ? (
          <View style={styles.rightIconWrapper}>{rightIcon}</View>
        ) : null}
      </View>

      {error ? (
        <Text style={styles.error} accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    marginBottom: spacing.lg,
  },
  label: {
    ...typography.label,
    fontSize: 13,
    lineHeight: 18,
    color: theme.textPrimary,
    marginBottom: spacing.xs,
  },
  labelDisabled: {
    color: theme.textDisabled,
  },
  inputWrapper: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: radius.md,
    backgroundColor: theme.surface,
    paddingHorizontal: spacing.md,
    minHeight: 46,
  },
  wrapperFocused: {
    borderColor: theme.primary,
  },
  wrapperError: {
    borderColor: theme.danger,
  },
  textInput: {
    flex: 1,
    ...typography.input,
    fontSize: 15,
    color: theme.textPrimary,
    paddingVertical: spacing.sm,
    paddingHorizontal: 0,
  },
  textInputDisabled: {
    color: theme.textDisabled,
  },
  leftIconWrapper: {
    marginRight: spacing.sm,
    alignItems: 'center',
    justifyContent: 'center',
  },
  rightIconWrapper: {
    marginLeft: spacing.sm,
    alignItems: 'center',
    justifyContent: 'center',
  },
  error: {
    ...typography.caption,
    color: theme.dangerText,
    marginTop: spacing.xs,
  },
})
