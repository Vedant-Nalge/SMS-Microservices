package com.sms.sender.service;

import com.sms.sender.kafka.SmsEventProducer;
import com.sms.sender.model.Models;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class SmsService {

    private final BlockListService blockListService;
    private final SmsVendorService vendorService;
    private final SmsEventProducer eventProducer;

    /**
     * Core send-SMS business flow:
     * 1. Check block list (Redis).
     * 2. Call mock vendor API.
     * 3. Publish Kafka event for the SMS Store.
     * 4. Return result to caller.
     */
    public Models.SendSmsResponse sendSms(Models.SendSmsRequest request) {
        String messageId = UUID.randomUUID().toString();
        Instant now = Instant.now();

        log.info("Processing SMS request: messageId={}, userId={}, phoneNumber={}",
                messageId, request.getUserId(), request.getPhoneNumber());

        // ── Step 1: Block-list check ─────────────────────────────────────────
        if (blockListService.isBlocked(request.getUserId())) {
            log.warn("SMS blocked for userId={}", request.getUserId());

            publishEvent(messageId, request, "BLOCKED", "User is on the block list", now);

            return Models.SendSmsResponse.builder()
                    .messageId(messageId)
                    .status("BLOCKED")
                    .userId(request.getUserId())
                    .phoneNumber(request.getPhoneNumber())
                    .errorMessage("User is on the block list")
                    .timestamp(now)
                    .build();
        }

        // ── Step 2: Call vendor API ──────────────────────────────────────────
        Models.VendorResult vendorResult;
        try {
            vendorResult = vendorService.send(request.getPhoneNumber(), request.getMessage());
        } catch (Exception e) {
            log.error("Unexpected error calling vendor for messageId={}: {}", messageId, e.getMessage(), e);
            vendorResult = Models.VendorResult.builder()
                    .success(false)
                    .status("FAILED")
                    .errorDetails("Vendor call threw exception: " + e.getMessage())
                    .build();
        }

        // ── Step 3: Publish Kafka event ──────────────────────────────────────
        publishEvent(messageId, request, vendorResult.getStatus(), vendorResult.getErrorDetails(), now);

        // ── Step 4: Build and return response ───────────────────────────────
        return Models.SendSmsResponse.builder()
                .messageId(messageId)
                .status(vendorResult.getStatus())
                .userId(request.getUserId())
                .phoneNumber(request.getPhoneNumber())
                .errorMessage(vendorResult.isSuccess() ? null : vendorResult.getErrorDetails())
                .timestamp(now)
                .build();
    }

    // ── Private helpers ──────────────────────────────────────────────────────

    private void publishEvent(String messageId,
                              Models.SendSmsRequest request,
                              String status,
                              String vendorResponse,
                              Instant sentAt) {
        try {
            Models.SmsEvent event = Models.SmsEvent.builder()
                    .messageId(messageId)
                    .userId(request.getUserId())
                    .phoneNumber(request.getPhoneNumber())
                    .message(request.getMessage())
                    .status(status)
                    .vendorResponse(vendorResponse)
                    .sentAt(sentAt)
                    .build();

            eventProducer.publishSmsEvent(event);
        } catch (Exception e) {
            // Log but don't fail the request — Kafka publish is best-effort here
            log.error("Failed to publish Kafka event for messageId={}: {}", messageId, e.getMessage(), e);
        }
    }
}
