package com.sms.sender.service;

import com.sms.sender.kafka.SmsEventProducer;
import com.sms.sender.model.Models;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class SmsServiceTest {

    @Mock private BlockListService blockListService;
    @Mock private SmsVendorService vendorService;
    @Mock private SmsEventProducer eventProducer;

    @InjectMocks private SmsService smsService;

    private Models.SendSmsRequest validRequest;

    @BeforeEach
    void setUp() {
        validRequest = Models.SendSmsRequest.builder()
                .userId("user-123")
                .phoneNumber("+919876543210")
                .message("Hello, World!")
                .build();
    }

    // ── Block-list tests ─────────────────────────────────────────────────────

    @Test
    @DisplayName("sendSms: returns BLOCKED when user is on block list")
    void sendSms_blocked() {
        when(blockListService.isBlocked("user-123")).thenReturn(true);

        Models.SendSmsResponse response = smsService.sendSms(validRequest);

        assertThat(response.getStatus()).isEqualTo("BLOCKED");
        assertThat(response.getErrorMessage()).containsIgnoringCase("block");
        verify(vendorService, never()).send(any(), any());
        verify(eventProducer).publishSmsEvent(argThat(e -> "BLOCKED".equals(e.getStatus())));
    }

    // ── Vendor success tests ─────────────────────────────────────────────────

    @Test
    @DisplayName("sendSms: returns SUCCESS when vendor call succeeds")
    void sendSms_vendorSuccess() {
        when(blockListService.isBlocked(any())).thenReturn(false);
        when(vendorService.send(any(), any())).thenReturn(
                Models.VendorResult.builder().success(true).status("SUCCESS").vendorMessageId("VND-ABC").build());

        Models.SendSmsResponse response = smsService.sendSms(validRequest);

        assertThat(response.getStatus()).isEqualTo("SUCCESS");
        assertThat(response.getMessageId()).isNotBlank();
        assertThat(response.getErrorMessage()).isNull();

        ArgumentCaptor<Models.SmsEvent> captor = ArgumentCaptor.forClass(Models.SmsEvent.class);
        verify(eventProducer).publishSmsEvent(captor.capture());
        assertThat(captor.getValue().getStatus()).isEqualTo("SUCCESS");
        assertThat(captor.getValue().getUserId()).isEqualTo("user-123");
    }

    // ── Vendor failure tests ─────────────────────────────────────────────────

    @Test
    @DisplayName("sendSms: returns FAILED when vendor call fails")
    void sendSms_vendorFailure() {
        when(blockListService.isBlocked(any())).thenReturn(false);
        when(vendorService.send(any(), any())).thenReturn(
                Models.VendorResult.builder().success(false).status("FAILED")
                        .errorDetails("delivery failure").build());

        Models.SendSmsResponse response = smsService.sendSms(validRequest);

        assertThat(response.getStatus()).isEqualTo("FAILED");
        assertThat(response.getErrorMessage()).isNotBlank();
        verify(eventProducer).publishSmsEvent(argThat(e -> "FAILED".equals(e.getStatus())));
    }

    @Test
    @DisplayName("sendSms: handles vendor exception gracefully")
    void sendSms_vendorThrowsException() {
        when(blockListService.isBlocked(any())).thenReturn(false);
        when(vendorService.send(any(), any())).thenThrow(new RuntimeException("connection timeout"));

        Models.SendSmsResponse response = smsService.sendSms(validRequest);

        assertThat(response.getStatus()).isEqualTo("FAILED");
        verify(eventProducer).publishSmsEvent(any());
    }

    // ── Kafka resilience test ────────────────────────────────────────────────

    @Test
    @DisplayName("sendSms: does not fail when Kafka publish throws")
    void sendSms_kafkaPublishFails_doesNotPropagateError() {
        when(blockListService.isBlocked(any())).thenReturn(false);
        when(vendorService.send(any(), any())).thenReturn(
                Models.VendorResult.builder().success(true).status("SUCCESS").vendorMessageId("VND-XYZ").build());
        doThrow(new RuntimeException("kafka unavailable")).when(eventProducer).publishSmsEvent(any());

        Models.SendSmsResponse response = smsService.sendSms(validRequest);

        // Kafka failure should not propagate — caller still gets vendor result
        assertThat(response.getStatus()).isEqualTo("SUCCESS");
    }
}
