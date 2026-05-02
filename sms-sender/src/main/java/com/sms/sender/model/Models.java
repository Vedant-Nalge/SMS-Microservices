package com.sms.sender.model;

import com.fasterxml.jackson.annotation.JsonFormat;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;

public final class Models {

    private Models() {}

    // ── Inbound request ──────────────────────────────────────────────────────

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SendSmsRequest {

        @NotBlank(message = "userId is required")
        private String userId;

        @NotBlank(message = "phoneNumber is required")
        @Pattern(regexp = "^\\+?[1-9]\\d{6,14}$", message = "phoneNumber must be a valid E.164 number")
        private String phoneNumber;

        @NotBlank(message = "message is required")
        @Size(max = 1600, message = "message must not exceed 1600 characters")
        private String message;
    }

    // ── Outbound response ─────────────────────────────────────────────────────

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SendSmsResponse {
        private String messageId;
        private String status;
        private String phoneNumber;
        private String userId;
        private String errorMessage;

        @JsonFormat(shape = JsonFormat.Shape.STRING)
        private Instant timestamp;
    }

    // ── Kafka event ───────────────────────────────────────────────────────────

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SmsEvent {
        private String messageId;
        private String userId;
        private String phoneNumber;
        private String message;
        private String status;
        private String vendorResponse;

        @JsonFormat(shape = JsonFormat.Shape.STRING)
        private Instant sentAt;
    }

    // ── Vendor result (internal) ──────────────────────────────────────────────

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class VendorResult {
        private boolean success;
        private String status;
        private String vendorMessageId;
        private String errorDetails;
    }
}
