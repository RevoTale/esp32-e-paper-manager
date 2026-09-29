# Exercise the actual checker and project policy; generated fixtures stay in build/.
file(MAKE_DIRECTORY "${OUTPUT_DIR}")
foreach(lines IN ITEMS 5 61)
    string(REPEAT "    ;\n" ${lines} body)
    set(source "${OUTPUT_DIR}/body-${lines}.c")
    file(WRITE "${source}" "void fixture(void) {\n${body}}\n")
    execute_process(COMMAND "${CLANG_TIDY}" "--config-file=${CONFIG}"
        "${source}" -- -x c -std=c11
        RESULT_VARIABLE result OUTPUT_VARIABLE output ERROR_VARIABLE error)
    if(lines EQUAL 5)
        if(NOT result EQUAL 0)
            message(FATAL_ERROR "Short function rejected: ${output}${error}")
        endif()
    elseif(NOT result EQUAL 1 OR
           NOT "${output}${error}" MATCHES "readability-function-size" OR
           "${output}${error}" MATCHES "clang-diagnostic-error")
        message(FATAL_ERROR "Long function not rejected by size rule: ${result}: ${output}${error}")
    endif()
endforeach()
