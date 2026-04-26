import smtplib


def test_constants():
    assert smtplib.SMTP_PORT == 25
    assert smtplib.SMTP_SSL_PORT == 465
    assert smtplib.LMTP_PORT == 2003
    print("constants ok")


def test_exception_hierarchy():
    # SMTPException is OSError
    assert issubclass(smtplib.SMTPException, OSError)
    assert issubclass(smtplib.SMTPException, Exception)
    # Direct subclasses of SMTPException
    assert issubclass(smtplib.SMTPServerDisconnected, smtplib.SMTPException)
    assert issubclass(smtplib.SMTPNotSupportedError, smtplib.SMTPException)
    assert issubclass(smtplib.SMTPRecipientsRefused, smtplib.SMTPException)
    assert issubclass(smtplib.SMTPResponseException, smtplib.SMTPException)
    # Subclasses of SMTPResponseException
    assert issubclass(smtplib.SMTPSenderRefused, smtplib.SMTPResponseException)
    assert issubclass(smtplib.SMTPDataError, smtplib.SMTPResponseException)
    assert issubclass(smtplib.SMTPConnectError, smtplib.SMTPResponseException)
    assert issubclass(smtplib.SMTPHeloError, smtplib.SMTPResponseException)
    assert issubclass(smtplib.SMTPAuthenticationError, smtplib.SMTPResponseException)
    # Everything is also SMTPException
    assert issubclass(smtplib.SMTPSenderRefused, smtplib.SMTPException)
    assert issubclass(smtplib.SMTPConnectError, smtplib.SMTPException)
    print("exception_hierarchy ok")


def test_smtp_response_exception_attrs():
    try:
        raise smtplib.SMTPResponseException(421, b'Service unavailable')
    except smtplib.SMTPResponseException as e:
        assert e.smtp_code == 421, repr(e.smtp_code)
        assert e.smtp_error == b'Service unavailable', repr(e.smtp_error)
    print("smtp_response_exception_attrs ok")


def test_smtp_sender_refused_attrs():
    try:
        raise smtplib.SMTPSenderRefused(550, b'Address rejected', 'sender@example.com')
    except smtplib.SMTPSenderRefused as e:
        assert e.smtp_code == 550, repr(e.smtp_code)
        assert e.smtp_error == b'Address rejected', repr(e.smtp_error)
        assert e.sender == 'sender@example.com', repr(e.sender)
    print("smtp_sender_refused_attrs ok")


def test_smtp_recipients_refused_attrs():
    recipients = {'a@b.com': (550, b'refused')}
    try:
        raise smtplib.SMTPRecipientsRefused(recipients)
    except smtplib.SMTPRecipientsRefused as e:
        assert e.recipients == recipients, repr(e.recipients)
    print("smtp_recipients_refused_attrs ok")


def test_exception_catch_as_oserror():
    try:
        raise smtplib.SMTPException("connection failed")
    except OSError as e:
        assert "connection failed" in str(e)
    try:
        raise smtplib.SMTPConnectError(421, b'unavailable')
    except OSError as e:
        pass
    try:
        raise smtplib.SMTPServerDisconnected("disconnected")
    except smtplib.SMTPException as e:
        assert "disconnected" in str(e)
    print("exception_catch_as_oserror ok")


def test_exception_distinct():
    try:
        raise smtplib.SMTPException("smtp error")
    except ValueError:
        assert False, "SMTPException should not be caught as ValueError"
    except smtplib.SMTPException:
        pass
    print("exception_distinct ok")


def test_smtp_construction():
    s = smtplib.SMTP()
    assert s.esmtp_features == {}, repr(s.esmtp_features)
    assert s.does_esmtp == False
    assert s.helo_resp is None
    assert s.ehlo_resp is None
    assert s.debuglevel == 0
    assert s.local_hostname is not None
    print("smtp_construction ok")


def test_smtp_set_debuglevel():
    s = smtplib.SMTP()
    assert s.debuglevel == 0
    s.set_debuglevel(1)
    assert s.debuglevel == 1
    s.set_debuglevel(0)
    assert s.debuglevel == 0
    print("smtp_set_debuglevel ok")


def test_smtp_has_extn():
    s = smtplib.SMTP()
    assert s.has_extn('SIZE') == False
    assert s.has_extn('AUTH') == False
    print("smtp_has_extn ok")


def test_smtp_close():
    s = smtplib.SMTP()
    s.close()
    print("smtp_close ok")


def test_smtp_context_manager():
    with smtplib.SMTP() as s:
        assert s is not None
    print("smtp_context_manager ok")


def test_class_hierarchy():
    assert issubclass(smtplib.SMTP_SSL, smtplib.SMTP)
    assert issubclass(smtplib.LMTP, smtplib.SMTP)
    print("class_hierarchy ok")


def test_module_exports():
    assert smtplib.SMTP is not None
    assert smtplib.SMTP_SSL is not None
    assert smtplib.LMTP is not None
    assert smtplib.SMTPException is not None
    assert smtplib.SMTPResponseException is not None
    assert smtplib.SMTPSenderRefused is not None
    assert smtplib.SMTPRecipientsRefused is not None
    assert smtplib.SMTPServerDisconnected is not None
    assert smtplib.SMTPConnectError is not None
    assert smtplib.SMTPHeloError is not None
    assert smtplib.SMTPDataError is not None
    assert smtplib.SMTPAuthenticationError is not None
    assert smtplib.SMTPNotSupportedError is not None
    print("module_exports ok")


test_constants()
test_exception_hierarchy()
test_smtp_response_exception_attrs()
test_smtp_sender_refused_attrs()
test_smtp_recipients_refused_attrs()
test_exception_catch_as_oserror()
test_exception_distinct()
test_smtp_construction()
test_smtp_set_debuglevel()
test_smtp_has_extn()
test_smtp_close()
test_smtp_context_manager()
test_class_hierarchy()
test_module_exports()
