package com.seaskyland.llm.workflow.core.config;

import org.apache.ibatis.type.BaseTypeHandler;
import org.apache.ibatis.type.JdbcType;
import org.apache.ibatis.type.MappedJdbcTypes;
import org.apache.ibatis.type.MappedTypes;

import java.sql.CallableStatement;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;

/**
 * Custom MyBatis TypeHandler for {@link java.util.Date} when using SQLite.
 *
 * <p>Problem: SQLite JDBC (3.41+) stores {@code Timestamp} values as epoch-millisecond numbers
 * (e.g. {@code 1772904625265}). When MyBatis reads them back via {@code getTimestamp()},
 * SQLite JDBC tries to parse that number as a formatted date string and throws
 * {@code "Error parsing time stamp"}.
 *
 * <p>Solution:
 * <ul>
 *   <li><b>Write</b> – always persist as an ISO-format string via {@code setString()},
 *       bypassing SQLite JDBC's {@code setTimestamp()} entirely.</li>
 *   <li><b>Read</b> – read as raw {@code String} via {@code getString()}, then try to parse
 *       as a formatted date first; if that fails, fall back to treating the value as
 *       epoch milliseconds (for rows already stored by previous versions).</li>
 * </ul>
 */
@MappedTypes({Date.class})
@MappedJdbcTypes(value = {JdbcType.TIMESTAMP, JdbcType.DATE, JdbcType.TIME}, includeNullJdbcType = true)
public class SQLiteDateTypeHandler extends BaseTypeHandler<Date> {

    private static final String[] READ_FORMATS = {
            "yyyy-MM-dd HH:mm:ss.SSS",
            "yyyy-MM-dd HH:mm:ss",
            "yyyy-MM-dd HH:mm",
            "yyyy-MM-dd"
    };

    private static final String WRITE_FORMAT = "yyyy-MM-dd HH:mm:ss";

    // -------------------------------------------------------------------------
    // Write
    // -------------------------------------------------------------------------

    @Override
    public void setNonNullParameter(PreparedStatement ps, int i, Date parameter, JdbcType jdbcType)
            throws SQLException {
        // Always store as a readable ISO string – avoids SQLite JDBC epoch-ms storage behaviour.
        ps.setString(i, new SimpleDateFormat(WRITE_FORMAT).format(parameter));
    }

    // -------------------------------------------------------------------------
    // Read
    // -------------------------------------------------------------------------

    @Override
    public Date getNullableResult(ResultSet rs, String columnName) throws SQLException {
        return parse(rs.getString(columnName));
    }

    @Override
    public Date getNullableResult(ResultSet rs, int columnIndex) throws SQLException {
        return parse(rs.getString(columnIndex));
    }

    @Override
    public Date getNullableResult(CallableStatement cs, int columnIndex) throws SQLException {
        return parse(cs.getString(columnIndex));
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    private Date parse(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        // Fallback: epoch milliseconds stored as a plain number string.
        try {
            return new Date(Long.parseLong(value.trim()));
        } catch (NumberFormatException ignored) {
            // Not a number – try formatted patterns below.
        }
        // Try ISO date-time string patterns.
        for (String fmt : READ_FORMATS) {
            try {
                return new SimpleDateFormat(fmt).parse(value);
            } catch (ParseException ignored) {
                // Try next pattern.
            }
        }
        return null;
    }
}
